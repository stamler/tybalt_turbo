package reports

import (
	"archive/zip"
	"bytes"
	"io"
	"slices"
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

func TestZipCacheLookupRequiresExactManifestForDuplicateHashes(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("NewFilesystem returned error: %v", err)
	}
	defer fsys.Close()

	const (
		collectionID  = "o1vpz1mm7qsfoyy"
		sourcePath    = "expense-duplicate/shared_receipt.pdf"
		zipCacheClass = "receipts_by_committed_week_ending"
		zipCacheKey   = "2026-05-01"
		sharedHash    = "same-hash"
	)

	if err := fsys.Upload([]byte("fixture receipt"), collectionID+"/"+sourcePath); err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	oneAttachment := []Attachment{
		{
			Id:          "expense-a",
			Filename:    "shared_receipt.pdf",
			ZipFilename: "Expense-A-2026_May_01-100-shared_receipt.pdf",
			SourcePath:  sourcePath,
			Sha256:      sharedHash,
		},
	}
	record, err := zipAttachments(app, oneAttachment, collectionID, zipCacheClass, zipCacheKey)
	if err != nil {
		t.Fatalf("zipAttachments returned error: %v", err)
	}

	cachedManifest, hasManifest, err := zipCacheManifestField(record, "manifest")
	if err != nil {
		t.Fatalf("zipCacheManifestField returned error: %v", err)
	}
	if !hasManifest {
		t.Fatal("expected zip cache record to store manifest")
	}
	wantManifest := []zipCacheManifestEntry{
		{
			CollectionID: "o1vpz1mm7qsfoyy",
			SourcePath:   "expense-duplicate/shared_receipt.pdf",
			ZipFilename:  "Expense-A-2026_May_01-100-shared_receipt.pdf",
			Sha256:       "same-hash",
		},
	}
	if !slices.Equal(cachedManifest, wantManifest) {
		t.Fatalf("manifest = %#v, want %#v", cachedManifest, wantManifest)
	}
	if want := attachmentManifest(oneAttachment, collectionID); !slices.Equal(cachedManifest, want) {
		t.Fatalf("manifest = %#v, want %#v", cachedManifest, want)
	}

	if hit, err := zipCacheLookup(app, zipCacheKey, zipCacheClass, oneAttachment, collectionID); err != nil {
		t.Fatalf("zipCacheLookup returned error: %v", err)
	} else if hit == nil {
		t.Fatal("expected exact manifest to hit cache")
	}

	twoAttachments := append([]Attachment{}, oneAttachment...)
	twoAttachments = append(twoAttachments, Attachment{
		Id:          "expense-b",
		Filename:    "shared_receipt.pdf",
		ZipFilename: "Expense-B-2026_May_01-250-shared_receipt.pdf",
		SourcePath:  sourcePath,
		Sha256:      sharedHash,
	})

	if hit, err := zipCacheLookup(app, zipCacheKey, zipCacheClass, twoAttachments, collectionID); err != nil {
		t.Fatalf("zipCacheLookup returned error: %v", err)
	} else if hit != nil {
		t.Fatal("expected duplicate hash with an additional zip entry to miss cache")
	}

	if _, err := zipAttachments(app, twoAttachments, collectionID, zipCacheClass, zipCacheKey); err != nil {
		t.Fatalf("zipAttachments with duplicate hash entries returned error: %v", err)
	}

	reorderedAttachments := []Attachment{twoAttachments[1], twoAttachments[0]}
	if hit, err := zipCacheLookup(app, zipCacheKey, zipCacheClass, reorderedAttachments, collectionID); err != nil {
		t.Fatalf("zipCacheLookup returned error: %v", err)
	} else if hit == nil {
		t.Fatal("expected reordered exact manifest to hit cache")
	}

	changedZipName := append([]Attachment{}, twoAttachments...)
	changedZipName[0].ZipFilename = "Expense-A-2026_May_01-999-shared_receipt.pdf"
	if hit, err := zipCacheLookup(app, zipCacheKey, zipCacheClass, changedZipName, collectionID); err != nil {
		t.Fatalf("zipCacheLookup returned error: %v", err)
	} else if hit != nil {
		t.Fatal("expected changed zip filename with same hash to miss cache")
	}
}
