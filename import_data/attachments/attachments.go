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

func MigrateAttachments(parquetPath string, collectionId string) {
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
		Attachment            *string `parquet:"name=attachment, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
		DestinationAttachment *string `parquet:"name=destination_attachment, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
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

	// Query Parquet file using DuckDB
	rows, err := db.QueryContext(ctx, `
		SELECT attachment, destination_attachment 
		FROM read_parquet(?) 
		WHERE attachment IS NOT NULL AND attachment != ''
	`, parquetPath)
	if err != nil {
		log.Fatalf("Failed to query %s with DuckDB: %v", parquetPath, err)
	}
	defer rows.Close()

	log.Printf("Reading attachments from %s using DuckDB...", parquetPath)
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
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating DuckDB query results: %v", err)
	}

	log.Printf("Found %d attachments to process using DuckDB.", len(attachmentInfos))

	// 4. for each row in the slice
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

		log.Printf("Processing attachment: GCS: gs://%s/%s -> S3: s3://%s/%s", os.Getenv("GCS_BUCKET_NAME"), gcsObjectPath, os.Getenv("AWS_S3_BUCKET_NAME"), s3ObjectKey)

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
		log.Printf("Successfully downloaded gs://%s/%s to %s", os.Getenv("GCS_BUCKET_NAME"), gcsObjectPath, localTempFilePath)

		//    2. upload the file to AWS S3 using the destination_attachment path
		fileToUpload, err := os.Open(localTempFilePath)
		if err != nil {
			log.Printf("ERROR: Failed to open temporary file %s for S3 upload: %v", localTempFilePath, err)
			os.Remove(localTempFilePath) // Attempt to clean up temp file
			continue
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
		log.Printf("Successfully uploaded %s to S3 bucket %s key %s", localTempFilePath, os.Getenv("AWS_S3_BUCKET_NAME"), s3ObjectKey)

		// Clean up the temporary file
		if err := os.Remove(localTempFilePath); err != nil {
			log.Printf("WARNING: Failed to remove temporary file %s: %v", localTempFilePath, err)
		}
	}

}
