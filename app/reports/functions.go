package reports

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
)

// convertToCSV takes a slice of dbx.NullStringMap and converts it into a CSV formatted string.
func convertToCSV(report []dbx.NullStringMap, headers []string) (string, error) {
	if len(report) == 0 {
		return "", nil // Return empty string for empty report
	}

	if headers == nil {
		// Collect all unique headers
		headerMap := make(map[string]struct{})
		for _, row := range report {
			for key := range row {
				headerMap[key] = struct{}{}
			}
		}
		headers = make([]string, 0, len(headerMap))
		for key := range headerMap {
			headers = append(headers, key)
		}
		sort.Strings(headers) // Sort headers for consistent column order
	}

	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	// Write header row
	if err := writer.Write(headers); err != nil {
		return "", err // Error writing header
	}

	// Write data rows
	record := make([]string, len(headers)) // Preallocate slice for performance
	for _, rowMap := range report {
		for i, header := range headers {
			if val, ok := rowMap[header]; ok && val.Valid {
				record[i] = val.String
			} else {
				record[i] = "" // Use empty string for NULL or missing values
			}
		}
		if err := writer.Write(record); err != nil {
			// It's possible rows have different numbers of columns if not careful,
			// but csv.Writer handles this by default. Still, good to check.
			return "", err // Error writing record
		}
	}

	writer.Flush() // Ensure all data is written to the builder

	// Check for errors encountered during flushing
	if err := writer.Error(); err != nil {
		return "", err
	}

	return builder.String(), nil
}

// zipAttachments takes a slice of Attachment and produces a zip archive of each
// file referenced by the source_path property giving it the corresponding
// filename from the filename property. It then creates a zip_cache record with
// the hashes of the attachments and the zip file. Finally, it returns the
// zip_cache record.
func zipAttachments(app core.App, report []Attachment, collectionId string, class string, key string) (*core.Record, error) {
	hashes := []string{}
	filenames := []string{}
	if len(report) == 0 {
		return nil, fmt.Errorf("no attachments to zip")
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Open filesystem access from within the pocketbase app
	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem: %w", err)
	}
	defer fsys.Close()

	for _, attachment := range report {
		fullPath := collectionId + "/" + attachment.SourcePath
		filenameInZip := attachment.Filename

		blob, err := fsys.GetReader(fullPath)
		if err != nil {
			// If a file can't be fetched, the original function's error handling implies failing the entire zip.
			// Consider logging and continuing for a more robust zip process if partial zips are acceptable.
			zipWriter.Close() // Attempt to clean up zipWriter
			return nil, fmt.Errorf("failed to get file %s: %w", fullPath, err)
		}

		// Create a new file in the zip archive
		zipEntry, err := zipWriter.Create(filenameInZip)
		if err != nil {
			blob.Close()      // Ensure blob is closed before returning
			zipWriter.Close() // Attempt to close before returning
			return nil, fmt.Errorf("failed to create zip entry for %s: %w", filenameInZip, err)
		}

		// Copy the file content to the zip entry
		_, err = io.Copy(zipEntry, blob)
		if err != nil {
			blob.Close()      // Ensure blob is closed before returning
			zipWriter.Close() // Attempt to close
			return nil, fmt.Errorf("failed to copy content of %s to zip: %w", attachment.SourcePath, err)
		}

		// Explicitly close the blob after successful copy
		if err_close := blob.Close(); err_close != nil {
			// An error on close after successful copy.
			// Depending on requirements, this could be logged or could fail the entire operation.
			// Propagating the error to be consistent with other error handling.
			zipWriter.Close()
			return nil, fmt.Errorf("failed to close file %s after copying: %w", attachment.SourcePath, err_close)
		}

		hashes = append(hashes, attachment.Sha256)
		filenames = append(filenames, attachment.Id+"/"+attachment.Filename)

		app.Logger().Debug(
			"File copied to zip",
			"sourcePath", attachment.SourcePath,
			"filenameInZip", filenameInZip,
		)
	}

	// Close the zip writer to finalize the archive
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	zipFile, err := filesystem.NewFileFromBytes(buf.Bytes(), class+"_"+key+".zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create file from bytes: %w", err)
	}

	zipCacheCollection, err := app.FindCollectionByNameOrId("zip_cache")
	if err != nil {
		return nil, fmt.Errorf("failed to find zip cache collection: %w", err)
	}

	// Try to load and update an existing zip_cache record
	var zipCacheRecord *core.Record
	zipCacheRecord, err = app.FindFirstRecordByFilter(
		"zip_cache",
		`key = {:key} && class = {:class}`,
		dbx.Params{"key": key, "class": class},
	)
	if err != nil {
		// Create a zip_cache record
		zipCacheRecord = core.NewRecord(zipCacheCollection)
		zipCacheRecord.Set("key", key)
		zipCacheRecord.Set("class", class)
		if slices.Contains(hashes, "") {
			zipCacheRecord.Set("hashes", []string{})
		} else {
			zipCacheRecord.Set("hashes", hashes)
		}
		zipCacheRecord.Set("zip", zipFile)
		zipCacheRecord.Set("filenames", filenames)
		err = app.Save(zipCacheRecord)
		if err != nil {
			return nil, fmt.Errorf("failed to save zip cache record: %w", err)
		}
	} else {
		// Update the existing zip_cache record
		zipCacheRecord.Set("zip", zipFile)
		zipCacheRecord.Set("filenames", filenames)
		if slices.Contains(hashes, "") {
			zipCacheRecord.Set("hashes", []string{})
		} else {
			zipCacheRecord.Set("hashes", hashes)
		}
		err = app.Save(zipCacheRecord)
		if err != nil {
			return nil, fmt.Errorf("failed to save zip cache record: %w", err)
		}
	}

	return zipCacheRecord, nil
}

// This function looks for a record in the zip_cache collection that matches the
// key and class. If it finds a record, it checks the hashes property of the
// record and verifies that every attachment in the attachments slice has a
// matching hash in the hashes property and that the length of the hashes
// property is the same as the length of the attachments slice. If the checks
// pass, it returns the record. Otherwise, it returns nil.
func zipCacheLookup(app core.App, key string, class string, attachments []Attachment) (*core.Record, error) {
	// The zip_cache collection has properties key, class, hashes (JSON array of strings), filenames (JSON array of strings), and zip (the zip file).
	// In most cases key will be a date string, but it could be anything.
	// The class will be the identifier of the class of zips that the zip is for.

	zipCacheRecord, err := app.FindFirstRecordByFilter(
		"zip_cache",
		`key = {:key} && class = {:class}`,
		dbx.Params{"key": key, "class": class},
	)
	if err != nil {
		// don't bother reporting the error, just treat it as a cache miss
		return nil, nil
	}

	// Load the hashes property into a slice of strings
	hashesJson := zipCacheRecord.Get("hashes").(types.JSONRaw)
	hashes := []string{}
	err = json.Unmarshal(hashesJson, &hashes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal hashes: %w", err)
	}

	if slices.Contains(hashes, "") || len(hashes) != len(attachments) {
		// If any of the hashes is an empty string or the length of hashes doesn't
		// match the length of the attachments, fallback to verifying the filenames
		// instead
		app.Logger().Debug("at least one hash is empty, falling back to filenames")
		filenamesJson := zipCacheRecord.Get("filenames").(types.JSONRaw)
		filenames := []string{}
		err = json.Unmarshal(filenamesJson, &filenames)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal filenames: %w", err)
		}
		if len(filenames) != len(attachments) {
			app.Logger().Debug("zip_cache miss for filename length mismatch")
			return nil, nil
		}
		for _, filename := range filenames {
			found := false
			for _, attachment := range attachments {
				if attachment.Id+"/"+attachment.Filename == filename {
					found = true
					break
				}
			}
			if !found {
				app.Logger().Debug("zip_cache miss for filename: " + filename)
				return nil, nil
			}
		}
	} else {
		// Otherwise, verify that each hash in the hashes slice is in the
		// attachments slice. If not, return nil.
		for _, hash := range hashes {
			found := false
			for _, attachment := range attachments {
				if attachment.Sha256 == hash {
					found = true
					break
				}
			}
			if !found {
				app.Logger().Debug("zip_cache miss for hash: " + hash)
				return nil, nil
			}
		}
	}

	// Return the zip cache record
	return zipCacheRecord, nil
}
