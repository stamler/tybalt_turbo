package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"
	"tybalt/errs"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var TimeTrackingNow = time.Now

var storageUnsafeFileNameChars = strings.NewReplacer(
	"#", "_",
	"?", "_",
)

// NormalizePendingFileNames rewrites uploaded file names for a field so the
// resulting storage keys don't contain URL fragment/query characters.
func NormalizePendingFileNames(record *core.Record, field string) {
	for _, file := range record.GetUnsavedFiles(field) {
		file.Name = storageUnsafeFileNameChars.Replace(file.Name)
	}
}

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

// EnsureActiveDivision verifies that the provided division id references an active
// division record. fieldName is used to attribute an error back to the caller.
func EnsureActiveDivision(app core.App, divisionID string, fieldName string) error {
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

// validateDivisionAllocatedToJob enforces the "job/division pair must be
// allocated" rule used by records that reference a job.
//
// Behavior:
//   - Empty job or division is treated as "not applicable" and returns no error.
//   - If allocation lookup fails, the function returns a job-scoped
//     "job_allocation_error" so callers can surface a system-style validation
//     failure without guessing.
//   - If the job has no allocation rows at all, the function returns a
//     job-scoped "job_no_allocations" error.
//   - If the division is not in the job's allocation set, the function returns
//     a division-scoped "division_not_allowed" error with the human-friendly
//     division code when available.
//
// Return contract:
//   - field name: where the caller should attach the error ("job" or "division")
//   - error: nil when the pair is valid or not applicable
func validateDivisionAllocatedToJob(app core.App, jobID string, divisionID string) (string, error) {
	if strings.TrimSpace(jobID) == "" || strings.TrimSpace(divisionID) == "" {
		return "", nil
	}

	var allocatedDivisions []string
	if err := app.DB().NewQuery(`
		SELECT division FROM job_time_allocations WHERE job = {:job}
	`).Bind(dbx.Params{"job": jobID}).Column(&allocatedDivisions); err != nil {
		return "job", validation.NewError(
			"job_allocation_error",
			"Error checking job allocations",
		)
	}

	if len(allocatedDivisions) == 0 {
		return "job", validation.NewError(
			"job_no_allocations",
			"This job has no division allocations configured",
		)
	}

	if slices.Contains(allocatedDivisions, divisionID) {
		return "", nil
	}

	divCode := divisionID
	if divRecord, divErr := app.FindRecordById("divisions", divisionID); divErr == nil && divRecord != nil {
		if code := strings.TrimSpace(divRecord.GetString("code")); code != "" {
			divCode = code
		}
	}

	return "division", validation.NewError(
		"division_not_allowed",
		fmt.Sprintf("Division %s is not allocated to this job", divCode),
	)
}

// shouldValidateJobDivisionAllocationOnRecord determines whether a hook should
// run allocation membership validation for a specific record write.
//
// Why this exists:
// Some legacy records may contain job/division pairs that predate strict
// allocation enforcement. We still want strict behavior for new data and for
// explicit job/division edits, but we don't want unrelated updates (for example
// comment/approval changes) to fail solely because old rows are imperfect.
//
// Rules:
//   - Create: always validate allocation.
//   - Update: validate only when job or division changed from persisted values.
//
// The helper first tries record.Original(); if unavailable, it falls back to
// reloading the current persisted record by ID.
func shouldValidateJobDivisionAllocationOnRecord(app core.App, record *core.Record) bool {
	if record == nil || record.IsNew() {
		return true
	}

	original := record.Original()
	if original == nil {
		loaded, err := app.FindRecordById(record.Collection().Name, record.Id)
		if err != nil || loaded == nil {
			return true
		}
		original = loaded
	}

	return strings.TrimSpace(record.GetString("job")) != strings.TrimSpace(original.GetString("job")) ||
		strings.TrimSpace(record.GetString("division")) != strings.TrimSpace(original.GetString("division"))
}

func validateAwardedProposalTimeTrackingWindow(app core.App, proposalRecord *core.Record, now time.Time) error {
	var projectAwardDates []string
	if err := app.DB().NewQuery(`
		SELECT project_award_date
		FROM jobs
		WHERE proposal = {:proposal}
		  AND project_award_date != ''
	`).Bind(dbx.Params{
		"proposal": proposalRecord.Id,
	}).Column(&projectAwardDates); err != nil {
		return validation.NewError("proposal_award_lookup_failed", "Error checking referencing project award dates")
	}

	if len(projectAwardDates) == 0 {
		return validation.NewError(
			"proposal_awarded_without_referencing_project",
			"Proposal is Awarded but has no referencing project yet; you cannot charge time to the proposal",
		)
	}

	earliestProjectAwardDate := strings.TrimSpace(slices.Min(projectAwardDates))

	awardDate, err := time.Parse(time.DateOnly, earliestProjectAwardDate)
	if err != nil {
		return validation.NewError("invalid_project_award_date", "Referenced project has an invalid project award date")
	}

	age := now.Sub(awardDate)
	if age < 0 {
		return validation.NewError("invalid_project_award_date", "Referenced project has a future project award date")
	}

	if age < 30*24*time.Hour {
		return nil
	}

	return validation.NewError(
		"proposal_awarded_more_than_30_days_ago",
		"Proposal was awarded more than 30 days in the past; use the referencing project instead",
	)
}

// validateJobAllowsTimeTracking applies the status rules for records that log
// time directly against jobs.
//
// Projects must be Active. Proposals are valid while they are still in flight,
// which in the current workflow means In Progress or Submitted. Awarded
// proposals get a short grace period based on the earliest referencing
// project's award date. We also allow legacy Active proposals so historical
// imports and older fixtures keep working.
func validateJobAllowsTimeTrackingAt(app core.App, jobRecord *core.Record, now time.Time) error {
	if jobRecord == nil {
		return validation.NewError("invalid_reference", "invalid job reference")
	}

	status := strings.TrimSpace(jobRecord.GetString("status"))
	if typeFromNumber(strings.TrimSpace(jobRecord.GetString("number"))) == jobTypeProposal {
		if status == "In Progress" || status == "Submitted" || status == "Active" {
			return nil
		}
		if status == "Awarded" {
			return validateAwardedProposalTimeTrackingWindow(app, jobRecord, now)
		}
		return validation.NewError("invalid_proposal_status", "Proposal status must be In Progress, Submitted, or recently Awarded")
	}

	if status != "Active" {
		return validation.NewError("not_active", "Job status must be Active")
	}

	return nil
}

func validateJobAllowsTimeTracking(app core.App, jobRecord *core.Record) error {
	return validateJobAllowsTimeTrackingAt(app, jobRecord, TimeTrackingNow())
}
