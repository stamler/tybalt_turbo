package attachments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"google.golang.org/api/option"
)

// writebackPrefix is the legacy GCS namespace used for Turbo-originated files.
// We keep this explicit so the importer can resolve both legacy and writeback
// paths without needing to know which system created the attachment.
const writebackPrefix = "Writeback/"

func gcsSourceCandidates(attachmentPath string) []string {
	// Build a prioritized list of GCS object paths to try for a given attachment.
	// Why:
	// - Historically, legacy expenses store attachments under "Expenses/{uid}/{hash}.{ext}".
	// - Turbo writeback copies attachments into legacy GCS under "Writeback/Expenses/{uid}/{hash}.{ext}".
	// - A clean export/augment normalizes parquet to the legacy path (no Writeback/),
	//   so attachmentPath should be legacy-style after you regenerate parquet.
	// - We still need to find the object regardless of where it actually lives in GCS
	//   (legacy vs writeback prefix).
	// Strategy:
	// - If attachmentPath already includes Writeback/ (unexpected with fresh parquet),
	//   try it first, then fall back to the trimmed legacy path.
	// - If attachmentPath is legacy-style (expected), try it first, then fall back to
	//   the Writeback/ prefixed variant.
	// This keeps the read path resilient without changing how destination S3 keys
	// are derived (those still come from destination_attachment).
	if attachmentPath == "" {
		return nil
	}
	if strings.HasPrefix(attachmentPath, writebackPrefix) {
		trimmed := strings.TrimPrefix(attachmentPath, writebackPrefix)
		if trimmed == "" {
			return []string{attachmentPath}
		}
		return []string{attachmentPath, trimmed}
	}
	return []string{attachmentPath, writebackPrefix + attachmentPath}
}

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
	gcsBucketName := os.Getenv("GCS_BUCKET_NAME")
	gcsBucket := gcsClient.Bucket(gcsBucketName)

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
	parquetS3Keys := make(map[string]bool) // ADDED: map to store all S3 keys defined in Parquet

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
		var attachment, destAttachment string
		if err := rows.Scan(&attachment, &destAttachment); err != nil {
			log.Printf("Warning: Failed to scan row from DuckDB result: %v. Skipping row.", err)
			continue
		}

		// Construct the potential S3 key to check against pre-fetched list
		if attachment != "" && destAttachment != "" {
			s3ObjectFullPath := collectionId + "/" + destAttachment
			parquetS3Keys[s3ObjectFullPath] = true // Mark this key as defined by Parquet

			// Check if this Parquet-defined object already exists in S3
			if _, exists := existingS3Objects[s3ObjectFullPath]; exists {
				// Object already exists in S3, so skip adding it to attachmentInfos
				skippedAttachments++ // We can count skipped items here as well
			} else {
				// Object is in Parquet but NOT in S3. Add to attachmentInfos for upload.
				// Both attachment (GCS path) and destAttachment (S3 key part) are guaranteed non-empty here.
				currentAttachment := Attachment{
					Attachment:            &attachment,
					DestinationAttachment: &destAttachment,
				}
				attachmentInfos = append(attachmentInfos, currentAttachment)
			}
		} else {
			// Log if essential parts for forming an S3 key or identifying a GCS source are missing
			log.Printf("Warning: Skipping row from Parquet due to missing GCS source ('%s') or S3 destination part ('%s').", attachment, destAttachment)
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

		// Object does not exist in our pre-fetched list, proceed with download and upload.

		//    1. download the attachment from Google Cloud Storage to the local file system
		// We now resolve attachments from either legacy or writeback namespaces:
		// - First candidate: the path in parquet (normalized, legacy-style).
		// - Second candidate: the same path under Writeback/.
		// This enables round-trip fidelity when attachments were only ever copied
		// into the Writeback/ prefix during Turbo writeback.
		var rc *storage.Reader
		var resolvedGcsPath string
		var readerErr error
		for _, candidate := range gcsSourceCandidates(gcsObjectPath) {
			// Try each candidate path in order; stop on the first one that exists.
			reader, err := gcsBucket.Object(candidate).NewReader(ctx)
			if err != nil {
				if errors.Is(err, storage.ErrObjectNotExist) {
					// Keep trying the next candidate when the object doesn't exist at
					// this prefix.
					continue
				}
				// For any other error (permissions, network, etc), capture it and abort
				// further attempts so we don't mask the underlying failure.
				readerErr = err
				log.Printf("ERROR: Failed to create reader for GCS object gs://%s/%s: %v", gcsBucketName, candidate, err)
				break
			}
			rc = reader
			resolvedGcsPath = candidate
			break
		}
		if rc == nil {
			if readerErr == nil {
				// Both legacy and writeback prefixes were checked and no object exists.
				// This is expected if the mirror hasn't synced yet, or if the attachment
				// reference is stale. We log and skip to keep the migration resilient.
				log.Printf("ERROR: GCS object not found at gs://%s/%s (also checked Writeback/ prefix)", gcsBucketName, gcsObjectPath)
			}
			continue
		}

		// Create a temporary local file
		// Use a subdirectory in the current directory to avoid cluttering the root
		tempDir := "./temp_attachments"
		if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
			log.Printf("ERROR: Failed to create temporary directory %s: %v", tempDir, err)
			continue
		}
		// Use the resolved path's base name for the temp file; this preserves the
		// correct extension when we had to fall back to Writeback/ prefixed objects.
		localTempFilePath := filepath.Join(tempDir, filepath.Base(resolvedGcsPath))
		localFile, err := os.Create(localTempFilePath)
		if err != nil {
			log.Printf("ERROR: Failed to create temporary file %s: %v", localTempFilePath, err)
			rc.Close() // Close GCS reader
			continue
		}

		if _, err := io.Copy(localFile, rc); err != nil {
			log.Printf("ERROR: Failed to download GCS object gs://%s/%s to %s: %v", gcsBucketName, resolvedGcsPath, localTempFilePath, err)
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

	// --- Deletion Phase: Remove S3 objects not found in the Parquet file ---
	log.Println("Starting S3 deletion phase for objects not in Parquet file...")
	objectsToDelete := []string{}
	// Iterate over all objects found in S3 for the given collectionId
	for s3KeyInBucket := range existingS3Objects {
		// Check if this S3 object (which includes collectionId prefix) is in our map of Parquet-defined keys
		if _, foundInParquet := parquetS3Keys[s3KeyInBucket]; !foundInParquet {
			// This S3 object was not listed in the Parquet source, so it should be deleted.
			objectsToDelete = append(objectsToDelete, s3KeyInBucket)
		}
	}

	if len(objectsToDelete) > 0 {
		log.Printf("Found %d S3 objects to delete (not present in Parquet specification under collectionId %s).", len(objectsToDelete), collectionId)
		deletedCount := 0
		for i, s3KeyToDelete := range objectsToDelete {
			// s3KeyToDelete already includes the collectionId prefix, e.g., "collectionId/objectName.txt"
			log.Printf("Deleting S3 object %d/%d: s3://%s/%s", i+1, len(objectsToDelete), os.Getenv("AWS_S3_BUCKET_NAME"), s3KeyToDelete)
			_, err := s3Svc.DeleteObject(&s3.DeleteObjectInput{
				Bucket: aws.String(os.Getenv("AWS_S3_BUCKET_NAME")),
				Key:    aws.String(s3KeyToDelete), // Key is the full path within the bucket
			})
			if err != nil {
				log.Printf("ERROR: Failed to delete S3 object %s: %v", s3KeyToDelete, err)
				// Depending on policy, you might choose to not proceed or stop on error
			} else {
				deletedCount++
				// Log for each successful deletion can be verbose, summary log is preferred.
				// log.Printf("Successfully deleted S3 object %s", s3KeyToDelete)
			}
		}
		log.Printf("Deletion phase complete. Successfully deleted %d of %d targeted S3 objects.", deletedCount, len(objectsToDelete))
	} else {
		log.Printf("No S3 objects found for deletion under collectionId %s (all existing S3 objects are specified in Parquet or the bucket was empty for this prefix).", collectionId)
	}
	// --- End Deletion Phase ---

}
