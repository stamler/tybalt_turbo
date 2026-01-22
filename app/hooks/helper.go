package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"tybalt/errs"

	"github.com/pocketbase/pocketbase/core"
)

// CalculateFileFieldHash computes the SHA256 hash of a file uploaded to a record field.
// Returns empty string if no file was uploaded for the field.
// Returns error if multiple files were uploaded or if there was an error reading the file.
func CalculateFileFieldHash(e *core.RecordRequestEvent, field string) (string, error) {
	// Get any files that have been uploaded for the field.
	files := e.Record.GetUnsavedFiles(field)

	// If the field is not present in the multipart form, or if it is present
	// but no actual files were uploaded for it (e.g., an empty file list).
	if len(files) == 0 {
		// No new file for this field in the current request.
		// Return empty string and no error, as there's nothing to hash.
		return "", nil
	}

	// If more than one file was uploaded for the field, this is an error,
	// as we expect only one file per field.
	if len(files) > 1 {
		return "", &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error processing file for field: " + field,
			Data: map[string]errs.CodeError{
				field: {
					Code:    "too_many_files",
					Message: "too many files uploaded for field " + field,
				},
			},
		}
	}

	// At this point, len(files) == 1. Get the first (and only) file.
	fileReader := files[0].Reader

	// open the file
	file, err := fileReader.Open()
	if err != nil {
		return "", &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "hook error opening file for field: " + field,
			Data: map[string]errs.CodeError{
				field: {
					Code:    "error_opening_file",
					Message: "error opening file for field " + field,
				},
			},
		}
	}
	defer file.Close()

	// calculate the hash of the file
	log.Println("calculating hash for", field)

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "hook error when calculating attachment hash",
			Data: map[string]errs.CodeError{
				field: {
					Code:    "error_calculating_hash",
					Message: "error calculating hash",
				},
			},
		}
	}

	// return the hash as a hex string
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ensureActiveDivision verifies that the provided division id references an active
// division record. fieldName is used to attribute an error back to the caller.
func ensureActiveDivision(app core.App, divisionID string, fieldName string) error {
	if divisionID == "" {
		return nil
	}

	division, err := app.FindRecordById("divisions", divisionID)
	if err != nil || division == nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "division lookup failed",
			Data: map[string]errs.CodeError{
				fieldName: {
					Code:    "invalid_division",
					Message: "specified division could not be found",
				},
			},
		}
	}

	if !division.GetBool("active") {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "division is inactive",
			Data: map[string]errs.CodeError{
				fieldName: {
					Code:    "not_active",
					Message: "specified division is inactive",
				},
			},
		}
	}

	return nil
}
