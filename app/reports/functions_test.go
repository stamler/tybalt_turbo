package reports

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"tybalt/internal/testseed"
)

func TestZipAttachmentsUsesFormattedZipFilename(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("NewFilesystem returned error: %v", err)
	}
	defer fsys.Close()

	const (
		collectionID   = "o1vpz1mm7qsfoyy"
		sourcePath     = "expense-123/original_receipt.pdf"
		zipEntryName   = "Expense-Doe,Jane-2026_Apr_13-123.45-abcd1234.pdf"
		zipCacheClass  = "receipts_by_committed_week_ending"
		zipCacheLookup = "2026-04-12"
	)

	if err := fsys.Upload([]byte("fixture receipt"), collectionID+"/"+sourcePath); err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	record, err := zipAttachments(app, []Attachment{
		{
			Id:          "expense-123",
			Filename:    "original_receipt.pdf",
			ZipFilename: zipEntryName,
			SourcePath:  sourcePath,
			Sha256:      "deadbeef",
		},
	}, collectionID, zipCacheClass, zipCacheLookup)
	if err != nil {
		t.Fatalf("zipAttachments returned error: %v", err)
	}

	zipPath := record.Collection().Id + "/" + record.Id + "/" + record.GetString("zip")
	reader, err := fsys.GetReader(zipPath)
	if err != nil {
		t.Fatalf("GetReader returned error: %v", err)
	}
	defer reader.Close()

	zipBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}

	archive, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("zip.NewReader returned error: %v", err)
	}

	if len(archive.File) != 1 {
		t.Fatalf("archive file count = %d, want 1", len(archive.File))
	}

	if got := archive.File[0].Name; got != zipEntryName {
		t.Fatalf("archive filename = %q, want %q", got, zipEntryName)
	}
}
