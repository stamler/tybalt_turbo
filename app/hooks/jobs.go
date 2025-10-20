package hooks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// ProcessJob enforces business rules for job creation and updates.
func ProcessJob(app core.App, e *core.RecordRequestEvent) error {
	jobRecord := e.Record

	if err := ensureOutstandingBalancePermission(app, e); err != nil {
		return err
	}

	if err := cleanJobOutstandingBalance(app, e); err != nil {
		return err
	}

	divisionsRaw := jobRecord.Get("divisions")

	var divisions []string
	switch v := divisionsRaw.(type) {
	case types.JSONRaw:
		if len(v) > 0 {
			if err := json.Unmarshal(v, &divisions); err != nil {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "division validation error",
					Data: map[string]errs.CodeError{
						"divisions": {
							Code:    "invalid_json",
							Message: "divisions must be an array of strings",
						},
					},
				}
			}
		}
	case []string:
		divisions = v
	case []any:
		divisions = make([]string, 0, len(v))
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "division validation error",
					Data: map[string]errs.CodeError{
						"divisions": {
							Code:    "invalid_json",
							Message: "divisions must be an array of strings",
						},
					},
				}
			}
			divisions = append(divisions, str)
		}
	case nil:
		// nothing provided
	default:
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "job divisions must be a JSON array",
			Data: map[string]errs.CodeError{
				"divisions": {
					Code:    "invalid_type",
					Message: "divisions must be a JSON array",
				},
			},
		}
	}

	for _, divisionID := range divisions {
		if err := ensureActiveDivision(app, divisionID, "divisions"); err != nil {
			return err
		}
	}

	// On create, derive the type of job while validating the job then generate a number
	if jobRecord.IsNew() {
		// First derive type while validating the job
		derivedType, derr := validateJob(app, e)
		if derr != nil {
			return derr
		}

		// Sub-job vs top-level number assignment based on parent relation
		parentRef := jobRecord.GetString("parent")
		if parentRef != "" {
			// parent existence and type already validated in validateJob; fetch and generate child number
			parentRec, _ := app.FindRecordById("jobs", parentRef)
			base := parentRec.GetString("number")
			childNumber, genErr := generateChildJobNumber(app, base)
			if genErr != nil {
				return &errs.HookError{
					Status:  http.StatusInternalServerError,
					Message: "error generating sub-job number",
					Data: map[string]errs.CodeError{
						"number": {Code: "error_generating_job_number", Message: genErr.Error()},
					},
				}
			}
			jobRecord.Set("number", childNumber)
		} else {
			// Top-level number
			jobNumber, genErr := generateTopLevelJobNumber(app, derivedType)
			if genErr != nil {
				return &errs.HookError{
					Status:  http.StatusInternalServerError,
					Message: "error generating job number",
					Data: map[string]errs.CodeError{
						"number": {Code: "error_generating_job_number", Message: genErr.Error()},
					},
				}
			}
			jobRecord.Set("number", jobNumber)
		}
		// done creating number
	} else {
		// We are updating an existing job
		// Validate job fields, cross-record constraints and derive type
		derivedType, err := validateJob(app, e)
		if err != nil {
			return err
		}
		// On update, ensure existing number implies a type that matches the payload
		impliedType := typeFromNumber(jobRecord.GetString("number"))
		if impliedType != derivedType {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "job number type conflict",
				Data: map[string]errs.CodeError{
					"number": {Code: "job_number_type_conflict", Message: "existing number implies a different job type"},
				},
			}
		}
	}

	// TODO: Follow up to confirm whether duplicate division ids can slip through here
	// and if so decide whether they should be rejected or automatically de-duplicated.

	return nil
}

func cleanJobOutstandingBalance(app core.App, e *core.RecordRequestEvent) error {
	jobRecord := e.Record
	outstandingBalance := jobRecord.GetFloat("outstanding_balance")

	originalRecord := jobRecord.Original()
	previousOutstandingBalance := 0.0
	hasOriginal := originalRecord != nil
	if hasOriginal {
		previousOutstandingBalance = originalRecord.GetFloat("outstanding_balance")
	}

	outstandingChanged := false
	switch {
	case !hasOriginal:
		outstandingChanged = outstandingBalance != 0
	default:
		outstandingChanged = outstandingBalance != previousOutstandingBalance
	}

	if outstandingChanged {
		jobRecord.Set("outstanding_balance_date", time.Now().Format("2006-01-02"))
	} else if hasOriginal {
		jobRecord.Set("outstanding_balance_date", originalRecord.Get("outstanding_balance_date"))
	}

	return nil
}

func ensureOutstandingBalancePermission(app core.App, e *core.RecordRequestEvent) error {
	jobRecord := e.Record
	originalRecord := jobRecord.Original()
	if originalRecord == nil {
		return nil
	}

	newOutstanding := jobRecord.GetFloat("outstanding_balance")
	oldOutstanding := originalRecord.GetFloat("outstanding_balance")
	if newOutstanding == oldOutstanding {
		return nil
	}

	if e.Auth == nil {
		return &errs.HookError{
			Status:  http.StatusForbidden,
			Message: "authentication required to edit outstanding balance",
			Data: map[string]errs.CodeError{
				"outstanding_balance": {
					Code:    "forbidden",
					Message: "authentication required",
				},
			},
		}
	}

	hasJobClaim, err := utilities.HasClaim(app, e.Auth, "job")
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "error checking jobs claim",
			Data: map[string]errs.CodeError{
				"outstanding_balance": {
					Code:    "claim_check_failed",
					Message: "unable to verify jobs claim",
				},
			},
		}
	}

	if hasJobClaim {
		return nil
	}

	hasPayablesClaim, err := utilities.HasClaim(app, e.Auth, "payables_admin")
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "error checking payables_admin claim",
			Data: map[string]errs.CodeError{
				"outstanding_balance": {
					Code:    "claim_check_failed",
					Message: "unable to verify payables_admin claim",
				},
			},
		}
	}

	if hasPayablesClaim {
		return nil
	}

	return &errs.HookError{
		Status:  http.StatusForbidden,
		Message: "insufficient permissions to edit outstanding balance",
		Data: map[string]errs.CodeError{
			"outstanding_balance": {
				Code:    "missing_claim",
				Message: "must have jobs or payables_admin claim",
			},
		},
	}
}

// --- Helper logic for jobs validation and numbering ---------------------------------

type jobType int

const (
	jobTypeProject jobType = iota
	jobTypeProposal
)

func (t jobType) String() string {
	if t == jobTypeProposal {
		return "proposal"
	}
	return "project"
}

var (
	baseNumberRegex  = regexp.MustCompile(`^(?:P)?\d{2}-\d{4}$`)
	childNumberRegex = regexp.MustCompile(`^(?:P)?\d{2}-\d{4}-\d{2}$`)
)

func isBaseNumber(s string) bool {
	return baseNumberRegex.MatchString(s)
}

// Produce the job type from a job number
func typeFromNumber(num string) jobType {
	n := num
	// strip child suffix if present
	if childNumberRegex.MatchString(n) {
		parts := strings.Split(n, "-")
		n = strings.Join(parts[:2], "-")
	}
	if strings.HasPrefix(n, "P") {
		return jobTypeProposal
	}
	return jobTypeProject
}

// validateJob enforces date/status rules, cross-record constraints and returns a derived type
func validateJob(app core.App, e *core.RecordRequestEvent) (jobType, error) {
	record := e.Record
	original := record.Original()
	isCreate := record.IsNew()

	// Extract fields
	proposalRef := record.GetString("proposal")
	parentRef := record.GetString("parent")
	projectAwardDate := record.GetString("project_award_date")
	proposalOpeningDate := record.GetString("proposal_opening_date")
	proposalSubmissionDueDate := record.GetString("proposal_submission_due_date")
	status := record.GetString("status")

	// Derive type: on create use explicit date configuration; on update infer from immutable number
	var derived jobType
	if isCreate {
		if proposalRef != "" {
			derived = jobTypeProject
		} else if projectAwardDate != "" && proposalOpeningDate == "" && proposalSubmissionDueDate == "" {
			derived = jobTypeProject
		} else if projectAwardDate == "" && proposalOpeningDate != "" && proposalSubmissionDueDate != "" {
			derived = jobTypeProposal
		} else {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "exactly one of project or proposal date configurations is required",
				Data: map[string]errs.CodeError{
					"project_award_date":           {Code: "invalid_date_configuration", Message: "provide project_award_date or both proposal dates"},
					"proposal_opening_date":        {Code: "invalid_date_configuration", Message: "provide both proposal dates or project_award_date"},
					"proposal_submission_due_date": {Code: "invalid_date_configuration", Message: "provide both proposal dates or project_award_date"},
				},
			}
		}
	} else {
		derived = typeFromNumber(original.GetString("number"))
	}

	// Enforce mutually exclusive date rules
	if isCreate {
		// create: strict requirements
		if derived == jobTypeProject {
			if projectAwardDate == "" || proposalOpeningDate != "" || proposalSubmissionDueDate != "" {
				return 0, &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "invalid dates for project",
					Data: map[string]errs.CodeError{
						"project_award_date":           {Code: "required_for_project", Message: "project_award_date is required"},
						"proposal_opening_date":        {Code: "not_permitted_for_project", Message: "proposal_opening_date must be empty for projects"},
						"proposal_submission_due_date": {Code: "not_permitted_for_project", Message: "proposal_submission_due_date must be empty for projects"},
					},
				}
			}
		} else { // proposal
			if projectAwardDate != "" || proposalOpeningDate == "" || proposalSubmissionDueDate == "" {
				return 0, &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "invalid dates for proposal",
					Data: map[string]errs.CodeError{
						"project_award_date":           {Code: "not_permitted_for_proposal", Message: "project_award_date must be empty for proposals"},
						"proposal_opening_date":        {Code: "required_for_proposal", Message: "proposal_opening_date is required"},
						"proposal_submission_due_date": {Code: "required_for_proposal", Message: "proposal_submission_due_date is required"},
					},
				}
			}
		}
	} else {
		// update: only enforce if any date field changed
		origAward := original.GetString("project_award_date")
		origOpen := original.GetString("proposal_opening_date")
		origDue := original.GetString("proposal_submission_due_date")
		changedAward := projectAwardDate != origAward
		changedOpen := proposalOpeningDate != origOpen
		changedDue := proposalSubmissionDueDate != origDue
		if changedAward || changedOpen || changedDue {
			// use effective values (new if provided, else original)
			effectiveAward := projectAwardDate
			if effectiveAward == "" {
				effectiveAward = origAward
			}
			effectiveOpen := proposalOpeningDate
			if effectiveOpen == "" {
				effectiveOpen = origOpen
			}
			effectiveDue := proposalSubmissionDueDate
			if effectiveDue == "" {
				effectiveDue = origDue
			}
			if derived == jobTypeProject {
				if effectiveAward == "" || effectiveOpen != "" || effectiveDue != "" {
					return 0, &errs.HookError{
						Status:  http.StatusBadRequest,
						Message: "invalid dates for project",
						Data: map[string]errs.CodeError{
							"project_award_date":           {Code: "required_for_project", Message: "project_award_date is required"},
							"proposal_opening_date":        {Code: "not_permitted_for_project", Message: "proposal_opening_date must be empty for projects"},
							"proposal_submission_due_date": {Code: "not_permitted_for_project", Message: "proposal_submission_due_date must be empty for projects"},
						},
					}
				}
			} else { // proposal
				if effectiveAward != "" || effectiveOpen == "" || effectiveDue == "" {
					return 0, &errs.HookError{
						Status:  http.StatusBadRequest,
						Message: "invalid dates for proposal",
						Data: map[string]errs.CodeError{
							"project_award_date":           {Code: "not_permitted_for_proposal", Message: "project_award_date must be empty for proposals"},
							"proposal_opening_date":        {Code: "required_for_proposal", Message: "proposal_opening_date is required"},
							"proposal_submission_due_date": {Code: "required_for_proposal", Message: "proposal_submission_due_date is required"},
						},
					}
				}
			}
		}
	}

	// Enforce status constraints
	if status != "" {
		if derived == jobTypeProject {
			if status == "Awarded" || status == "Not Awarded" {
				return 0, &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "invalid status for project",
					Data: map[string]errs.CodeError{
						"status": {Code: "invalid_status_for_type", Message: "projects may be Active, Closed or Cancelled"},
					},
				}
			}
		} else { // proposal
			if status == "Closed" {
				return 0, &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "invalid status for proposal",
					Data: map[string]errs.CodeError{
						"status": {Code: "invalid_status_for_type", Message: "proposals may be Active, Awarded, Not Awarded or Cancelled"},
					},
				}
			}
		}
	}

	// Cross-record constraints when referencing a proposal (creating/updating a project that points to a proposal)
	if proposalRef != "" {
		// ensure referenced job exists
		refRec, err := app.FindRecordById("jobs", proposalRef)
		if err != nil || refRec == nil {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "invalid referenced proposal",
				Data: map[string]errs.CodeError{
					"proposal": {Code: "invalid_reference", Message: "referenced proposal not found"},
				},
			}
		}
		// referenced must be a proposal (by number prefix)
		if typeFromNumber(refRec.GetString("number")) != jobTypeProposal {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "referenced job is not a proposal",
				Data: map[string]errs.CodeError{
					"proposal": {Code: "invalid_reference_type", Message: "referenced job must be a proposal"},
				},
			}
		}
		// referenced status must be Awarded; if Active → prompt-able error; Not Awarded/Cancelled → reject
		refStatus := refRec.GetString("status")
		if refStatus == "Active" {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "proposal must be awarded",
				Data: map[string]errs.CodeError{
					"proposal": {Code: "proposal_not_awarded", Message: "proposal must be set to Awarded", Data: map[string]string{"proposal_id": proposalRef}},
				},
			}
		}
		if refStatus == "Not Awarded" || refStatus == "Cancelled" || refStatus == "Closed" {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "proposal has invalid status",
				Data: map[string]errs.CodeError{
					"proposal": {Code: "referenced_proposal_invalid_status", Message: "proposal must be Awarded to reference"},
				},
			}
		}
		if refStatus != "Awarded" {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "proposal must be awarded",
				Data: map[string]errs.CodeError{
					"proposal": {Code: "proposal_not_awarded", Message: "proposal must be set to Awarded", Data: map[string]string{"proposal_id": proposalRef}},
				},
			}
		}

		// enforce single-project-per-proposal
		currentID := ""
		if original != nil {
			currentID = original.Id
		}
		existing, err := app.FindRecordsByFilter("jobs", "proposal = {:pid}", "-created", 2, 0, dbx.Params{"pid": proposalRef})
		if err == nil {
			for _, rec := range existing {
				if rec.Id != currentID {
					return 0, &errs.HookError{
						Status:  http.StatusBadRequest,
						Message: "proposal already referenced by another project",
						Data: map[string]errs.CodeError{
							"proposal": {Code: "proposal_already_referenced", Message: "only one project may reference a proposal"},
						},
					}
				}
			}
		}
	}

	// Sub-job parent constraints: if parent is set, parent must exist and must be a project (not a proposal)
	if parentRef != "" {
		parentRec, err := app.FindRecordById("jobs", parentRef)
		if err != nil || parentRec == nil {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "invalid parent",
				Data: map[string]errs.CodeError{
					"parent": {Code: "invalid_parent", Message: "specified parent job not found"},
				},
			}
		}
		// parent must be a project (cannot create sub-jobs under proposals)
		if typeFromNumber(parentRec.GetString("number")) == jobTypeProposal {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "proposals cannot have sub-jobs",
				Data: map[string]errs.CodeError{
					"parent": {Code: "proposal_cannot_have_subjobs", Message: "cannot create sub-job under a proposal"},
				},
			}
		}

		// enforce same client as parent
		parentClient := parentRec.GetString("client")
		childClient := record.GetString("client")
		if parentClient == "" {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "parent missing client",
				Data: map[string]errs.CodeError{
					"client": {Code: "parent_missing_client", Message: "parent job has no client"},
				},
			}
		}
		if childClient != "" && childClient != parentClient {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "sub-job must have same client as parent",
				Data: map[string]errs.CodeError{
					"client": {Code: "client_mismatch_with_parent", Message: "client must match parent"},
				},
			}
		}
		// write the parent's client to the record to guarantee consistency
		record.Set("client", parentClient)
	}

	// Update-time number/type consistency: if updating, implied type from number must match derived
	if original != nil {
		implied := typeFromNumber(original.GetString("number"))
		if implied != derived {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "job number type conflict",
				Data: map[string]errs.CodeError{
					"number": {Code: "job_number_type_conflict", Message: "existing number implies a different job type"},
				},
			}
		}
	}

	return derived, nil
}

func generateTopLevelJobNumber(app core.App, t jobType) (string, error) {
	yy := time.Now().Year() % 100
	var prefix string
	if t == jobTypeProposal {
		prefix = fmt.Sprintf("P%02d-", yy)
	} else {
		prefix = fmt.Sprintf("%02d-", yy)
	}

	// Fetch a batch of candidates ordered descending and find the highest base number
	records, err := app.FindRecordsByFilter("jobs", `number ~ {:prefix}`, "-number", 50, 0, dbx.Params{"prefix": prefix + "%"})
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	lastNumber := 0
	for _, r := range records {
		n := r.GetString("number")
		// consider only base numbers matching prefix + 4 digits
		if strings.HasPrefix(n, prefix) && baseNumberRegex.MatchString(n) {
			suf := strings.TrimPrefix(n, prefix)
			if v, convErr := strconv.Atoi(suf); convErr == nil {
				if v > lastNumber {
					lastNumber = v
				}
			}
		}
	}

	// Generate next unique number
	for i := lastNumber + 1; i <= 9999; i++ {
		candidate := fmt.Sprintf("%s%04d", prefix, i)
		existing, findErr := app.FindFirstRecordByFilter("jobs", "number = {:n}", dbx.Params{"n": candidate})
		if findErr != nil && findErr != sql.ErrNoRows {
			return "", findErr
		}
		if existing == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("unable to generate a unique job number")
}

func generateChildJobNumber(app core.App, base string) (string, error) {
	if !isBaseNumber(base) {
		return "", fmt.Errorf("invalid base number")
	}

	// Find highest child suffix for this base
	children, err := app.FindRecordsByFilter("jobs", `number ~ {:p}`, "-number", 10, 0, dbx.Params{"p": base + "-%"})
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	next := 1
	if len(children) > 0 {
		last := children[0].GetString("number")
		// last should end with -XX
		parts := strings.Split(last, "-")
		if len(parts) >= 3 {
			suf := parts[len(parts)-1]
			if v, convErr := strconv.Atoi(suf); convErr == nil {
				next = v + 1
			}
		}
	}
	if next > 99 {
		return "", fmt.Errorf("maximum number of sub-jobs reached (99) for %s", base)
	}
	candidate := fmt.Sprintf("%s-%02d", base, next)
	existing, findErr := app.FindFirstRecordByFilter("jobs", "number = {:n}", dbx.Params{"n": candidate})
	if findErr != nil && findErr != sql.ErrNoRows {
		return "", findErr
	}
	if existing != nil {
		return "", fmt.Errorf("generated sub-job number already exists: %s", candidate)
	}
	return candidate, nil
}
