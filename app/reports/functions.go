package reports

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
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

// zipAttachments takes a slice of dbx.NullStringMap and produces a zip archive
// of each file referenced by the source_path property giving it the
// corresponding filename from the filename property.
func zipAttachments(app core.App, report []dbx.NullStringMap, collectionId string) ([]byte, error) {
	if len(report) == 0 {
		return nil, nil // Return nil for empty report
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Open filesystem access from within the pocketbase app
	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem: %w", err)
	}
	defer fsys.Close()

	for _, rowMap := range report {
		sourcePathVal, sourcePathOk := rowMap["source_path"]
		filenameVal, filenameOk := rowMap["filename"]

		if !sourcePathOk || !sourcePathVal.Valid || !filenameOk || !filenameVal.Valid {
			// Skip if essential fields are missing or NULL
			// Or, return an error:
			// return nil, fmt.Errorf("missing 'source_path' or 'filename' in a row")
			continue
		}

		sourcePath := sourcePathVal.String
		fullPath := collectionId + "/" + sourcePath
		filenameInZip := filenameVal.String

		blob, err := fsys.GetFile(fullPath)
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
			return nil, fmt.Errorf("failed to copy content of %s to zip: %w", sourcePath, err)
		}

		// Explicitly close the blob after successful copy
		if err_close := blob.Close(); err_close != nil {
			// An error on close after successful copy.
			// Depending on requirements, this could be logged or could fail the entire operation.
			// Propagating the error to be consistent with other error handling.
			zipWriter.Close()
			return nil, fmt.Errorf("failed to close file %s after copying: %w", sourcePath, err_close)
		}

		app.Logger().Debug(
			"File copied to zip",
			"sourcePath", sourcePath,
			"filenameInZip", filenameInZip,
		)
	}

	// Close the zip writer to finalize the archive
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	return buf.Bytes(), nil
}
