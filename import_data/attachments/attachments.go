package attachments

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"google.golang.org/api/option"
)

func MigrateAttachments(parquetPath string, sourceColumn string, destinationColumn string, collectionId string) {
	fmt.Println("Importing attachments from GCS to S3...")

	// --- Migrate attachments from Google Cloud Storage to the local file system,
	// renaming them correctly, then uploading them to AWS S3.

	// 1. setup the Google Cloud Storage client
	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("GCS_SERVICE_ACCOUNT_JSON")))
	if err != nil {
		log.Fatalf("Failed to create Google Cloud Storage client: %v", err)
	}
	defer gcsClient.Close()
	_ = gcsClient.Bucket(os.Getenv("GCS_BUCKET_NAME"))

	// 2. setup the AWS S3 client
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(os.Getenv("AWS_REGION")),
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	s3Svc := s3.New(sess)
	// --- Pre-fetch existing S3 objects ---
	// This is done *before* reading from Parquet to filter attachmentInfos upfront.
	existingS3Objects := make(map[string]bool)
	log.Printf("Fetching existing objects from S3 bucket %s under prefix %s/", os.Getenv("AWS_S3_BUCKET_NAME"), collectionId)
	var continuationToken *string
	for {
		listObjectsInput := &s3.ListObjectsV2Input{
			Bucket:            aws.String(os.Getenv("AWS_S3_BUCKET_NAME")),
			Prefix:            aws.String(collectionId + "/"),
			ContinuationToken: continuationToken,
		}
		listObjectsOutput, err := s3Svc.ListObjectsV2(listObjectsInput)
		if err != nil {
			log.Fatalf("ERROR: Failed to list objects in S3 bucket %s under prefix %s/: %v", os.Getenv("AWS_S3_BUCKET_NAME"), collectionId, err)
		}

		for _, object := range listObjectsOutput.Contents {
			existingS3Objects[*object.Key] = true
		}

		if !*listObjectsOutput.IsTruncated {
			break // No more objects to list
		}
		continuationToken = listObjectsOutput.NextContinuationToken
	}
	log.Printf("Found %d existing objects in S3 under prefix %s/ for pre-filtering.", len(existingS3Objects), collectionId)
	// --- End S3 Pre-fetch ---

	// 3. from parquet file, `SELECT attachment, destination_attachment FROM data WHERE attachment IS NOT NULL` into a slice of structs
	type Attachment struct {
		Attachment            *string `parquet:"attachment"`
		DestinationAttachment *string `parquet:"destination_attachment"`
	}

	var attachmentInfos []Attachment
	// Initialize skippedAttachments here, before it's used in the Parquet processing loop
	skippedAttachments := 0

	// Setup DuckDB
	// The empty string for the DSN typically means an in-memory database for DuckDB.
	// The driver registers itself under the name "duckdb" upon import.
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatalf("Failed to open DuckDB database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil { // Good practice to ping the database
		log.Fatalf("Failed to ping DuckDB: %v", err)
	}

	// Get total count for progress
	var totalRows int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM read_parquet(?)
		WHERE %s IS NOT NULL AND %s != ''
	`, sourceColumn, sourceColumn) // Use Sprintf to safely inject column names
	err = db.QueryRowContext(ctx, countQuery, parquetPath).Scan(&totalRows)
	if err != nil {
		log.Fatalf("Failed to query total count from %s with DuckDB: %v", parquetPath, err)
	}
	log.Printf("Counted %d attachment rows in the Parquet file.", totalRows)

	// Query Parquet file using DuckDB
	// Construct the query string using fmt.Sprintf to allow dynamic column names
	// IMPORTANT: Ensure sourceColumn and destinationColumn are validated or come from a trusted source
	// to prevent SQL injection if they were user-provided in a different context.
	// Here, they are function parameters, which is generally safer.
	query := fmt.Sprintf(`
		SELECT %s as attachment, %s as destination_attachment
		FROM read_parquet(?)
		WHERE %s IS NOT NULL AND %s != ''
	`, sourceColumn, destinationColumn, sourceColumn, sourceColumn)

	rows, err := db.QueryContext(ctx, query, parquetPath)
	if err != nil {
		log.Fatalf("Failed to query %s with DuckDB: %v", parquetPath, err)
	}
	defer rows.Close()

	log.Printf("Loading attachment rows from %s...", parquetPath)
	processedRows := 0
	for rows.Next() {
		var a Attachment
		var attachment, destAttachment string
		if err := rows.Scan(&attachment, &destAttachment); err != nil {
			log.Printf("Warning: Failed to scan row from DuckDB result: %v. Skipping row.", err)
			continue
		}

		// Construct the potential S3 key to check against pre-fetched list
		if attachment != "" && destAttachment != "" {
			s3ObjectFullPath := collectionId + "/" + destAttachment
			if _, exists := existingS3Objects[s3ObjectFullPath]; exists {
				// Object already exists in S3, so skip adding it to attachmentInfos
				skippedAttachments++ // We can count skipped items here as well
				continue
			}
		}

		if attachment != "" {
			a.Attachment = &attachment
		}
		if destAttachment != "" {
			a.DestinationAttachment = &destAttachment
		}
		if a.Attachment != nil { // Only add if it's a valid attachment and not skipped
			attachmentInfos = append(attachmentInfos, a)
		}
		processedRows++
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating DuckDB query results: %v", err)
	}

	log.Printf("Found %d attachments missing from S3.", len(attachmentInfos))

	// 4. for each row in the slice (attachmentInfos now only contains items NOT in S3)
	uploadedAttachments := 0
	// skippedAttachments is now largely handled during attachmentInfos construction,
	// and initialized before the parquet reading loop.
	totalAttachmentsToUpload := len(attachmentInfos) // This is now the count of *new* items to upload

	for _, info := range attachmentInfos {
		if info.Attachment == nil || *info.Attachment == "" {
			log.Println("Skipping row with missing GCS attachment path (should not happen if filtered correctly).")
			continue
		}
		if info.DestinationAttachment == nil || *info.DestinationAttachment == "" {
			log.Println("Skipping row with missing S3 destination attachment path (should not happen if filtered correctly).")
			continue
		}

		gcsObjectPath := *info.Attachment
		s3ObjectKey := *info.DestinationAttachment // s3ObjectFullPath was used for check, use s3ObjectKey for PutObject's Key
		s3ObjectFullPathForUpload := collectionId + "/" + s3ObjectKey

		// The check for S3 existence is no longer needed here as attachmentInfos is pre-filtered.
		// if _, exists := existingS3Objects[s3ObjectFullPath]; exists {
		// // Object exists, skip everything for this attachment
		// skippedAttachments++
		// log.Printf("S3 Progress: %d uploaded + %d skipped / %d total (Object %s already exists)", uploadedAttachments, skippedAttachments, totalAttachmentsToUpload, s3ObjectFullPath)
		// continue // Skip to the next attachment
		// }

		// Object does not exist in our pre-fetched list, proceed with download and upload.

		//    1. download the attachment from Google Cloud Storage to the local file system
		gcsObject := gcsClient.Bucket(os.Getenv("GCS_BUCKET_NAME")).Object(gcsObjectPath)
		rc, err := gcsObject.NewReader(ctx)
		if err != nil {
			log.Printf("ERROR: Failed to create reader for GCS object gs://%s/%s: %v", os.Getenv("GCS_BUCKET_NAME"), gcsObjectPath, err)
			continue
		}

		// Create a temporary local file
		// Use a subdirectory in the current directory to avoid cluttering the root
		tempDir := "./temp_attachments"
		if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
			log.Printf("ERROR: Failed to create temporary directory %s: %v", tempDir, err)
			continue
		}
		localTempFilePath := filepath.Join(tempDir, filepath.Base(gcsObjectPath))
		localFile, err := os.Create(localTempFilePath)
		if err != nil {
			log.Printf("ERROR: Failed to create temporary file %s: %v", localTempFilePath, err)
			rc.Close() // Close GCS reader
			continue
		}

		if _, err := io.Copy(localFile, rc); err != nil {
			log.Printf("ERROR: Failed to download GCS object gs://%s/%s to %s: %v", os.Getenv("GCS_BUCKET_NAME"), gcsObjectPath, localTempFilePath, err)
			rc.Close()
			localFile.Close()
			os.Remove(localTempFilePath) // Attempt to clean up temp file
			continue
		}
		rc.Close()
		localFile.Close()

		//    2. upload the file to AWS S3 using the destination_attachment path
		fileToUpload, err := os.Open(localTempFilePath)
		if err != nil {
			log.Printf("ERROR: Failed to open temporary file %s for S3 upload: %v", localTempFilePath, err)
			os.Remove(localTempFilePath) // Attempt to clean up temp file
			continue
		}

		// Check if the object already exists in S3 using the pre-fetched list
		// s3ObjectFullPath := collectionId + "/" + s3ObjectKey // Defined earlier
		// if _, exists := existingS3Objects[s3ObjectFullPath]; exists {
		// Object exists, clean up temp file and skip upload
		// skippedAttachments++
		// log.Printf("S3 Progress: %d uploaded + %d skipped / %d total (Object %s already exists)", uploadedAttachments, skippedAttachments, totalAttachmentsToUpload, s3ObjectFullPath)
		// fileToUpload.Close()
		// os.Remove(localTempFilePath) // Clean up temp file
		// continue
		// }
		// Object does not exist in our pre-fetched list, proceed with upload attempt.
		// The original HeadObject call and its error handling logic below can be removed or commented out.

		// Check if the object already exists in S3
		// headObjectInput := &s3.HeadObjectInput{
		// 	Bucket: aws.String(os.Getenv("AWS_S3_BUCKET_NAME")),
		// 	Key:    aws.String(collectionId + "/" + s3ObjectKey),
		// }

		// _, err = s3Svc.HeadObject(headObjectInput)
		// if err == nil {
		// 	// Object exists, clean up temp file and skip upload
		// 	skippedAttachments++
		// 	log.Printf("S3 Progress: %d uploaded + %d skipped / %d total (Object %s already exists)", uploadedAttachments, skippedAttachments, totalAttachmentsToUpload, s3ObjectFullPath)
		// 	fileToUpload.Close()
		// 	os.Remove(localTempFilePath) // Clean up temp file
		// 	continue
		// } else {
		// 	// Check if the error is because the object was not found
		// 	if aerr, ok := err.(awserr.Error); ok {
		// 		if aerr.Code() != s3.ErrCodeNoSuchKey && aerr.Code() != "NotFound" {
		// 			// An unexpected error occurred with HeadObject, other than "NotFound"
		// 			log.Printf("ERROR: Failed to check S3 object existence for key %s: %v. Proceeding with upload attempt.", s3ObjectKey, err)
		// 			// Depending on policy, you might choose to not proceed or handle differently
		// 		}
		// 		// If err is "NotFound" or s3.ErrCodeNoSuchKey, we continue to PutObject
		// 	} else {
		// 		// Non-AWS error, log it and proceed with upload attempt cautiously
		// 		log.Printf("ERROR: Unexpected error type checking S3 object existence for key %s: %v. Proceeding with upload attempt.", s3ObjectKey, err)
		// 	}
		// }

		_, err = s3Svc.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(os.Getenv("AWS_S3_BUCKET_NAME")),
			Key:    aws.String(s3ObjectFullPathForUpload), // Use s3ObjectFullPathForUpload here
			Body:   fileToUpload,
		})
		fileToUpload.Close() // Close the file before attempting to remove it

		if err != nil {
			log.Printf("ERROR: Failed to upload %s to S3 bucket %s key %s: %v", localTempFilePath, os.Getenv("AWS_S3_BUCKET_NAME"), s3ObjectFullPathForUpload, err)
			os.Remove(localTempFilePath) // Attempt to clean up temp file
			continue
		}

		// Clean up the temporary file
		if err := os.Remove(localTempFilePath); err != nil {
			log.Printf("WARNING: Failed to remove temporary file %s: %v", localTempFilePath, err)
		}
		uploadedAttachments++
		// Add the newly uploaded object to our map so we don't try to re-upload if it appears again in attachmentInfos (edge case)
		// This is less critical now as attachmentInfos is pre-filtered, but good for robustness if an item somehow wasn't filtered
		existingS3Objects[s3ObjectFullPathForUpload] = true
		log.Printf("S3 Progress: %d uploaded %d total", uploadedAttachments, totalAttachmentsToUpload)
	}

}
