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
	"github.com/aws/aws-sdk-go/aws/awserr"
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
	_ = s3Svc // Placeholder to use s3Svc, remove when used in step 4

	// 3. from parquet file, `SELECT attachment, destination_attachment FROM data WHERE attachment IS NOT NULL` into a slice of structs
	type Attachment struct {
		Attachment            *string `parquet:"attachment"`
		DestinationAttachment *string `parquet:"destination_attachment"`
	}

	var attachmentInfos []Attachment

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
	log.Printf("Found %d total attachments to potentially process.", totalRows)

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

	log.Printf("Reading attachments from %s using DuckDB...", parquetPath)
	processedRows := 0
	for rows.Next() {
		var a Attachment
		var attachment, destAttachment string
		if err := rows.Scan(&attachment, &destAttachment); err != nil {
			log.Printf("Warning: Failed to scan row from DuckDB result: %v. Skipping row.", err)
			continue
		}
		if attachment != "" {
			a.Attachment = &attachment
		}
		if destAttachment != "" {
			a.DestinationAttachment = &destAttachment
		}
		if a.Attachment != nil {
			attachmentInfos = append(attachmentInfos, a)
		}
		processedRows++
		if totalRows > 0 {
			log.Printf("Progress: Processed %d/%d attachments from Parquet.", processedRows, totalRows)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating DuckDB query results: %v", err)
	}

	log.Printf("Found %d attachments to process using DuckDB.", len(attachmentInfos))

	// 4. for each row in the slice
	uploadedAttachments := 0
	skippedAttachments := 0
	totalAttachmentsToUpload := len(attachmentInfos)
	for _, info := range attachmentInfos {
		if info.Attachment == nil || *info.Attachment == "" {
			log.Println("Skipping row with missing GCS attachment path.")
			continue
		}
		if info.DestinationAttachment == nil || *info.DestinationAttachment == "" {
			log.Println("Skipping row with missing S3 destination attachment path.")
			continue
		}

		gcsObjectPath := *info.Attachment
		s3ObjectKey := *info.DestinationAttachment

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

		// Check if the object already exists in S3
		headObjectInput := &s3.HeadObjectInput{
			Bucket: aws.String(os.Getenv("AWS_S3_BUCKET_NAME")),
			Key:    aws.String(collectionId + "/" + s3ObjectKey),
		}

		_, err = s3Svc.HeadObject(headObjectInput)
		if err == nil {
			// Object exists, clean up temp file and skip upload
			skippedAttachments++
			log.Printf("S3 Progress: %d uploaded + %d skipped / %d total", uploadedAttachments, skippedAttachments, totalAttachmentsToUpload)
			fileToUpload.Close()
			os.Remove(localTempFilePath) // Clean up temp file
			continue
		} else {
			// Check if the error is because the object was not found
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() != s3.ErrCodeNoSuchKey && aerr.Code() != "NotFound" {
					// An unexpected error occurred with HeadObject, other than "NotFound"
					log.Printf("ERROR: Failed to check S3 object existence for key %s: %v. Proceeding with upload attempt.", s3ObjectKey, err)
					// Depending on policy, you might choose to not proceed or handle differently
				}
				// If err is "NotFound" or s3.ErrCodeNoSuchKey, we continue to PutObject
			} else {
				// Non-AWS error, log it and proceed with upload attempt cautiously
				log.Printf("ERROR: Unexpected error type checking S3 object existence for key %s: %v. Proceeding with upload attempt.", s3ObjectKey, err)
			}
		}

		_, err = s3Svc.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(os.Getenv("AWS_S3_BUCKET_NAME")),
			Key:    aws.String(collectionId + "/" + s3ObjectKey),
			Body:   fileToUpload,
		})
		fileToUpload.Close() // Close the file before attempting to remove it

		if err != nil {
			log.Printf("ERROR: Failed to upload %s to S3 bucket %s key %s: %v", localTempFilePath, os.Getenv("AWS_S3_BUCKET_NAME"), s3ObjectKey, err)
			os.Remove(localTempFilePath) // Attempt to clean up temp file
			continue
		}

		// Clean up the temporary file
		if err := os.Remove(localTempFilePath); err != nil {
			log.Printf("WARNING: Failed to remove temporary file %s: %v", localTempFilePath, err)
		}
		uploadedAttachments++
		log.Printf("S3 Progress: %d uploaded + %d skipped / %d total", uploadedAttachments, skippedAttachments, totalAttachmentsToUpload)
	}

}
