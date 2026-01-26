package hooks

import (
	"database/sql"
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
)

func ProcessJob(app core.App, e *core.RecordRequestEvent) error {

	return ProcessJobCore(app, e.Record, e.Auth)
}

// ProcessJobCore enforces business rules for job creation and updates.
// It operates on plain records rather than RecordRequestEvent, making it reusable
// from both PocketBase hooks and custom API endpoints.
func ProcessJobCore(app core.App, jobRecord *core.Record, authRecord *core.Record) error {
	// Check if job editing is enabled via app_config
	enabled, err := utilities.IsJobsEditingEnabled(app)
	if err != nil {
		return err
	}
	if !enabled {
		return utilities.ErrJobsEditingDisabled
	}

	// Cancelled proposals are terminal - reject all modifications
	if !jobRecord.IsNew() {
		original := jobRecord.Original()
		if typeFromNumber(original.GetString("number")) == jobTypeProposal {
			if original.GetString("status") == "Cancelled" {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "cancelled proposals cannot be modified",
					Data: map[string]errs.CodeError{
						"global": {Code: "cancelled_proposal_immutable", Message: "this proposal has been cancelled and cannot be modified"},
					},
				}
			}
		}
	}

	if err := ensureOutstandingBalancePermission(app, jobRecord, authRecord); err != nil {
		return err
	}

	// Caveat: cleanJob runs before validateJob. If cleanJob normalizes fields
	// (e.g., clears authorizing_document/client_po/client_reference_number on proposals
	// or updates outstanding_balance_date), a request intending to change only "status"
	// may appear to have other changes. If this causes status-only updates to be treated
	// as full updates, either exclude such normalized fields from the change detection
	// in validateJob or move the status-only check earlier in this function.
	// Normalize fields that depend on the job type (project vs proposal)
	if err := cleanJob(app, jobRecord); err != nil {
		return err
	}

	// On create, derive the type of job while validating the job then generate a number
	if jobRecord.IsNew() {
		// First derive type while validating the job
		derivedType, derr := validateJob(app, jobRecord)
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
						"global": {Code: "error_generating_job_number", Message: genErr.Error()},
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
						"global": {Code: "error_generating_job_number", Message: genErr.Error()},
					},
				}
			}
			jobRecord.Set("number", jobNumber)
		}
		// done creating number
	} else {
		// We are updating an existing job
		// Validate job fields, cross-record constraints and derive type
		_, err := validateJob(app, jobRecord)
		if err != nil {
			return err
		}
	}

	// TODO: Follow up to confirm whether duplicate division ids can slip through here
	// and if so decide whether they should be rejected or automatically de-duplicated.

	// On update, if any field changed, mark the record as no longer imported.
	// This ensures locally-modified jobs get written back to the legacy system.
	if !jobRecord.IsNew() {
		utilities.MarkImportedFalseIfChanged(jobRecord)
	}

	return nil
}

func cleanJobOutstandingBalance(jobRecord *core.Record) error {
	outstandingBalance := jobRecord.GetFloat("outstanding_balance")

	originalRecord := jobRecord.Original()
	isCreate := jobRecord.IsNew()
	previousOutstandingBalance := originalRecord.GetFloat("outstanding_balance")

	var outstandingChanged bool
	if isCreate {
		outstandingChanged = outstandingBalance != 0
	} else {
		outstandingChanged = outstandingBalance != previousOutstandingBalance
	}

	if outstandingChanged {
		jobRecord.Set("outstanding_balance_date", time.Now().Format("2006-01-02"))
	} else if !isCreate {
		jobRecord.Set("outstanding_balance_date", originalRecord.Get("outstanding_balance_date"))
	}

	return nil
}

func ensureOutstandingBalancePermission(app core.App, jobRecord *core.Record, authRecord *core.Record) error {
	// Detect creates based on the record state, not on the external "original" pointer.
	if jobRecord.IsNew() {
		return nil
	}
	originalRecord := jobRecord.Original()

	newOutstanding := jobRecord.GetFloat("outstanding_balance")
	oldOutstanding := originalRecord.GetFloat("outstanding_balance")
	if newOutstanding == oldOutstanding {
		return nil
	}

	if authRecord == nil {
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

	hasJobClaim, err := utilities.HasClaim(app, authRecord, "job")
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

	hasPayablesClaim, err := utilities.HasClaim(app, authRecord, "payables_admin")
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

// validateProposalDateOrder ensures the proposal submission due date is on or
// after the proposal opening date. Both dates must be non-empty strings in
// "YYYY-MM-DD" format.
func validateProposalDateOrder(openingDate, submissionDueDate string) error {
	opening, err := time.Parse("2006-01-02", openingDate)
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "invalid proposal opening date format",
			Data: map[string]errs.CodeError{
				"proposal_opening_date": {Code: "invalid_date_format", Message: "proposal_opening_date must be in YYYY-MM-DD format"},
			},
		}
	}
	due, err := time.Parse("2006-01-02", submissionDueDate)
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "invalid proposal submission due date format",
			Data: map[string]errs.CodeError{
				"proposal_submission_due_date": {Code: "invalid_date_format", Message: "proposal_submission_due_date must be in YYYY-MM-DD format"},
			},
		}
	}
	if due.Before(opening) {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "proposal submission due date must be on or after opening date",
			Data: map[string]errs.CodeError{
				"proposal_submission_due_date": {Code: "date_order_invalid", Message: "submission due date must be on or after opening date"},
			},
		}
	}
	return nil
}

// isProposalRecord determines if the current record should be treated as a
// proposal for cleaning/validation purposes at this stage of processing.
//
//   - On update, the job type is inferred from the immutable job number.
//   - On create, we infer proposal when the proposal date pair is provided and
//     project_award_date is empty (mirrors validateJob create-time derivation).
func isProposalRecord(record *core.Record) bool {
	if !record.IsNew() {
		return typeFromNumber(record.Original().GetString("number")) == jobTypeProposal
	}
	// Create-time inference (aligns with validateJob create-time logic for proposals)
	projectAwardDate := record.GetString("project_award_date")
	proposalOpeningDate := record.GetString("proposal_opening_date")
	proposalSubmissionDueDate := record.GetString("proposal_submission_due_date")
	return projectAwardDate == "" && proposalOpeningDate != "" && proposalSubmissionDueDate != ""
}

// cleanJob removes or normalizes fields that are not applicable based on the
// job type. In particular, proposals must not carry authorizing_document,
// client_po, or client_reference_number.
func cleanJob(app core.App, record *core.Record) error {
	isProposal := isProposalRecord(record)

	// Normalize client_po by trimming whitespace
	trimmedClientPO := strings.TrimSpace(record.GetString("client_po"))
	if record.GetString("client_po") != trimmedClientPO {
		record.Set("client_po", trimmedClientPO)
	}

	if !isProposal {
		// Projects should NOT have these proposal-specific fields
		record.Set("proposal_value", 0)
		record.Set("time_and_materials", false)

		// If authorizing_document != PO, clear any provided client_po
		if record.GetString("authorizing_document") != "PO" && record.GetString("client_po") != "" {
			record.Set("client_po", "")
		}

		// Centralize outstanding balance normalization for projects only
		if err := cleanJobOutstandingBalance(record); err != nil {
			return err
		}
	} else {
		// Proposals should NOT have these project-specific fields
		record.Set("authorizing_document", "")
		record.Set("client_po", "")
		record.Set("client_reference_number", "")
		record.Set("outstanding_balance", 0)
		record.Set("outstanding_balance_date", "")
	}
	return nil
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
	// baseNumberRegex matches top-level job numbers (not sub-jobs).
	// Format: [P]YY-NNN or [P]YY-NNNN
	//   - Optional "P" prefix for proposals
	//   - 2-digit year (e.g., "25" for 2025)
	//   - 3 or 4 digit job sequence number
	// Legacy jobs use 3 digits (e.g., "25-123"), newer jobs use 4 digits (e.g., "25-0123").
	// This regex is used by isBaseNumber() to determine if a job can have sub-jobs created under it.
	// Sub-jobs (e.g., "25-123-01") will NOT match this pattern, which prevents nested sub-jobs.
	baseNumberRegex = regexp.MustCompile(`^(?:P)?\d{2}-\d{3,4}$`)

	// childNumberRegex matches first-level sub-job numbers.
	// Format: [P]YY-NNN-SS or [P]YY-NNNN-SS
	//   - Base job number (3-4 digits)
	//   - 1 or 2 digit sub-job suffix
	// Used by typeFromNumber() to strip the sub-job suffix when determining job type.
	// Note: Nested sub-jobs (e.g., "25-123-01-1") exist in legacy data but are not matched here.
	// This is intentional - typeFromNumber() still works correctly because it only needs to
	// check if the number starts with "P" to determine the type, and that check works on any format.
	childNumberRegex = regexp.MustCompile(`^(?:P)?\d{2}-\d{3,4}-\d{1,2}$`)

	locationPlusCodeRegex = regexp.MustCompile(`^[23456789CFGHJMPQRVWX]{8}\+[23456789CFGHJMPQRVWX]{2,3}$`)
)

// isBaseNumber returns true if s is a valid base (top-level) job number.
// This is used to validate that a parent job can have sub-jobs created under it.
// Only base jobs can be parents - sub-jobs cannot have their own sub-jobs.
// Examples:
//   - "25-123" → true (legacy 3-digit format)
//   - "25-0123" → true (current 4-digit format)
//   - "P25-0123" → true (proposal)
//   - "25-0123-01" → false (sub-job, cannot be a parent)
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

func validateJob(app core.App, record *core.Record) (jobType, error) {
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

	// Allow status-only updates to pass without tripping other validations.
	// This compensates for relaxed update rules while preserving status constraints.
	if !isCreate {
		changedOtherField := false
		// Check if any field other than "status", "updated", or "created" changed; if none did, treat as status-only update.
		for _, fieldName := range record.Collection().Fields.FieldNames() {
			if fieldName == "status" || fieldName == "updated" || fieldName == "created" {
				continue // Skip the status field itself and auto-date fields
			}
			if fmt.Sprintf("%v", record.Get(fieldName)) != fmt.Sprintf("%v", original.Get(fieldName)) {
				changedOtherField = true
				break
			}
		}
		if !changedOtherField {
			newStatus := status
			oldStatus := original.GetString("status")
			if newStatus != "" && newStatus != oldStatus {
				if derived == jobTypeProject {
					// Projects: allowed statuses are Active, Closed, Cancelled
					if newStatus == "Awarded" || newStatus == "Not Awarded" || newStatus == "Submitted" || newStatus == "In Progress" || newStatus == "No Bid" {
						return 0, &errs.HookError{
							Status:  http.StatusBadRequest,
							Message: "invalid status for project",
							Data: map[string]errs.CodeError{
								"status": {Code: "invalid_status_for_type", Message: "projects may be Active, Closed or Cancelled"},
							},
						}
					}
				} else { // proposal
					// Proposals: allowed statuses are In Progress, Submitted, Awarded, Not Awarded, Cancelled, No Bid
					// Disallowed: Active, Closed
					if newStatus == "Active" || newStatus == "Closed" {
						return 0, &errs.HookError{
							Status:  http.StatusBadRequest,
							Message: "invalid status for proposal",
							Data: map[string]errs.CodeError{
								"status": {Code: "invalid_status_for_type", Message: "proposals may be In Progress, Submitted, Awarded, Not Awarded, Cancelled or No Bid"},
							},
						}
					}
					// For Submitted, Awarded, Not Awarded: require proposal_value > 0 OR time_and_materials = true
					if newStatus == "Submitted" || newStatus == "Awarded" || newStatus == "Not Awarded" {
						proposalValue := record.GetInt("proposal_value")
						timeAndMaterials := record.GetBool("time_and_materials")
						if proposalValue <= 0 && !timeAndMaterials {
							return 0, &errs.HookError{
								Status:  http.StatusBadRequest,
								Message: "proposal value or time and materials required",
								Data: map[string]errs.CodeError{
									"status": {Code: "value_required_for_status", Message: "proposals with status Submitted, Awarded, or Not Awarded must have a proposal value or be marked as time and materials"},
								},
							}
						}
					}
					// For No Bid or Cancelled: require a client_note with matching status_change_to
					if newStatus == "No Bid" || newStatus == "Cancelled" {
						jobID := record.Id
						hasNote, err := jobHasClientNoteForStatus(app, jobID, newStatus)
						if err != nil {
							return 0, &errs.HookError{
								Status:  http.StatusInternalServerError,
								Message: "failed to check for client notes",
								Data: map[string]errs.CodeError{
									"status": {Code: "note_check_failed", Message: "unable to verify client notes exist"},
								},
							}
						}
						if !hasNote {
							return 0, &errs.HookError{
								Status:  http.StatusBadRequest,
								Message: "a comment is required to set this status",
								Data: map[string]errs.CodeError{
									"status": {Code: "comment_required_for_status", Message: "a comment must be added before setting status to " + newStatus},
								},
							}
						}
					}
				}
			}
			return derived, nil
		}
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
			// Validate that submission due date is on or after opening date
			if err := validateProposalDateOrder(proposalOpeningDate, proposalSubmissionDueDate); err != nil {
				return 0, err
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
				// Validate that submission due date is on or after opening date
				if err := validateProposalDateOrder(effectiveOpen, effectiveDue); err != nil {
					return 0, err
				}
			}
		}
	}

	// All jobs must have a valid location (schema rules may be relaxed, enforce here)
	loc := record.GetString("location")
	if loc == "" || !locationPlusCodeRegex.MatchString(loc) {
		return 0, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "invalid or missing location",
			Data: map[string]errs.CodeError{
				"location": {Code: "invalid_or_missing", Message: "location (Plus Code) is required"},
			},
		}
	}

	// All jobs must have a branch
	if record.GetString("branch") == "" {
		return 0, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "branch is required",
			Data: map[string]errs.CodeError{
				"branch": {Code: "required", Message: "branch is required"},
			},
		}
	}

	// Enforce authorizing_document/client_po rules for projects (validation only).
	// Normalization is handled in cleanJob earlier.
	if derived == jobTypeProject {
		authorizingDocument := record.GetString("authorizing_document")
		if authorizingDocument == "" {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "authorizing document is required",
				Data: map[string]errs.CodeError{
					"authorizing_document": {Code: "required", Message: "authorizing_document is required"},
				},
			}
		}
		if authorizingDocument == "PO" {
			clientPO := record.GetString("client_po")
			if len(clientPO) <= 2 {
				return 0, &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "client PO must be at least 3 characters when authorizing document is PO",
					Data: map[string]errs.CodeError{
						"client_po": {Code: "client_po_min_length", Message: "client_po must be at least 3 characters when authorizing_document is PO"},
					},
				}
			}
		}
	}

	// Enforce status constraints
	if status != "" {
		if derived == jobTypeProject {
			// Projects: allowed statuses are Active, Closed, Cancelled
			if status == "Awarded" || status == "Not Awarded" || status == "Submitted" || status == "In Progress" || status == "No Bid" {
				return 0, &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "invalid status for project",
					Data: map[string]errs.CodeError{
						"status": {Code: "invalid_status_for_type", Message: "projects may be Active, Closed or Cancelled"},
					},
				}
			}
		} else { // proposal
			// Proposals: allowed statuses are In Progress, Submitted, Awarded, Not Awarded, Cancelled, No Bid
			// Disallowed: Active, Closed
			if status == "Active" || status == "Closed" {
				return 0, &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "invalid status for proposal",
					Data: map[string]errs.CodeError{
						"status": {Code: "invalid_status_for_type", Message: "proposals may be In Progress, Submitted, Awarded, Not Awarded, Cancelled or No Bid"},
					},
				}
			}
			// New proposals can only be "In Progress" or "Submitted" (other statuses require
			// the job ID to exist for comment requirements or represent later workflow states)
			if isCreate && status != "In Progress" && status != "Submitted" {
				return 0, &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "new proposals must start as In Progress or Submitted",
					Data: map[string]errs.CodeError{
						"status": {Code: "invalid_status_for_new_proposal", Message: "new proposals can only have status In Progress or Submitted"},
					},
				}
			}
			// For Submitted, Awarded, Not Awarded: require proposal_value > 0 OR time_and_materials = true
			if status == "Submitted" || status == "Awarded" || status == "Not Awarded" {
				proposalValue := record.GetInt("proposal_value")
				timeAndMaterials := record.GetBool("time_and_materials")
				if proposalValue <= 0 && !timeAndMaterials {
					return 0, &errs.HookError{
						Status:  http.StatusBadRequest,
						Message: "proposal value or time and materials required",
						Data: map[string]errs.CodeError{
							"proposal_value": {Code: "value_required_for_status", Message: "proposals with status Submitted, Awarded, or Not Awarded must have a proposal value or be marked as time and materials"},
						},
					}
				}
			}
			// For No Bid or Cancelled: require a client_note with matching status_change_to (only on updates)
			if !isCreate && (status == "No Bid" || status == "Cancelled") {
				oldStatus := original.GetString("status")
				// Only check when actually transitioning to this status
				if oldStatus != status {
					jobID := record.Id
					hasNote, err := jobHasClientNoteForStatus(app, jobID, status)
					if err != nil {
						return 0, &errs.HookError{
							Status:  http.StatusInternalServerError,
							Message: "failed to check for client notes",
							Data: map[string]errs.CodeError{
								"status": {Code: "note_check_failed", Message: "unable to verify client notes exist"},
							},
						}
					}
					if !hasNote {
						return 0, &errs.HookError{
							Status:  http.StatusBadRequest,
							Message: "a comment is required to set this status",
							Data: map[string]errs.CodeError{
								"status": {Code: "comment_required_for_status", Message: "a comment must be added before setting status to " + status},
							},
						}
					}
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
		// referenced status must be Awarded; if In Progress/Submitted → prompt-able error; Not Awarded/Cancelled/No Bid → reject
		refStatus := refRec.GetString("status")
		if refStatus == "In Progress" || refStatus == "Submitted" {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "proposal must be awarded",
				Data: map[string]errs.CodeError{
					"proposal": {Code: "proposal_not_awarded", Message: "proposal must be set to Awarded", Data: map[string]string{"proposal_id": proposalRef}},
				},
			}
		}
		if refStatus == "Not Awarded" || refStatus == "Cancelled" || refStatus == "No Bid" {
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

	// Client/contact consistency (moved from rules into hooks): if contact is set, it must belong to client
	contactRef := record.GetString("contact")
	if contactRef != "" {
		clientRef := record.GetString("client")
		if clientRef == "" {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "contact requires a client",
				Data: map[string]errs.CodeError{
					"client": {Code: "required_with_contact", Message: "client is required when contact is set"},
				},
			}
		}
		contactRec, err := app.FindRecordById("client_contacts", contactRef)
		if err != nil || contactRec == nil {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "invalid contact",
				Data: map[string]errs.CodeError{
					"contact": {Code: "invalid_reference", Message: "specified contact not found"},
				},
			}
		}
		if contactRec.GetString("client") != clientRef {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "contact does not belong to client",
				Data: map[string]errs.CodeError{
					"contact": {Code: "contact_client_mismatch", Message: "contact must belong to the selected client"},
				},
			}
		}
	}

	// Update-time number/type consistency: only enforce on update
	if !isCreate {
		implied := typeFromNumber(original.GetString("number"))
		if implied != derived {
			return 0, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "job number type conflict",
				Data: map[string]errs.CodeError{
					"global": {Code: "job_number_type_conflict", Message: "existing number implies a different job type"},
				},
			}
		}
	}

	// Validate that manager and alternate_manager are active users
	if err := validateManagersAreActive(app, record); err != nil {
		return 0, err
	}

	return derived, nil
}

// jobHasClientNoteForStatus checks if a client_note exists for the given job
// with job_status_changed_to matching the target status.
// This is used to enforce the requirement that a comment must be added before
// setting a proposal's status to "No Bid" or "Cancelled".
func jobHasClientNoteForStatus(app core.App, jobID string, targetStatus string) (bool, error) {
	note, err := app.FindFirstRecordByFilter("client_notes", "job = {:jobID} && job_status_changed_to = {:status}", dbx.Params{"jobID": jobID, "status": targetStatus})
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return note != nil, nil
}

// validateManagersAreActive checks that the manager and alternate_manager (if set)
// have admin_profiles records with active = true. Inactive users cannot be assigned
// as managers on jobs.
func validateManagersAreActive(app core.App, record *core.Record) error {
	managerUID := record.GetString("manager")
	altManagerUID := record.GetString("alternate_manager")

	// Check manager (required field, so should always have a value)
	if managerUID != "" {
		active, err := utilities.IsUserActive(app, managerUID)
		if err != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "failed to check manager active status",
			}
		}
		if !active {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "manager must be an active user",
				Data: map[string]errs.CodeError{
					"manager": {Code: "manager_not_active", Message: "the selected manager is not an active user"},
				},
			}
		}
	}

	// Check alternate_manager (optional field)
	if altManagerUID != "" {
		active, err := utilities.IsUserActive(app, altManagerUID)
		if err != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "failed to check alternate manager active status",
			}
		}
		if !active {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "alternate manager must be an active user",
				Data: map[string]errs.CodeError{
					"alternate_manager": {Code: "alternate_manager_not_active", Message: "the selected alternate manager is not an active user"},
				},
			}
		}
	}

	return nil
}

// generateTopLevelJobNumber generates the next available job number for a given job type.
//
// # Job Number Formats
//
// Job numbers exist in two formats:
//   - Legacy 3-digit: "26-015" (regular) or "P26-015" (proposal)
//   - Current 4-digit: "26-0015" (regular) or "P26-0015" (proposal)
//
// Starting in 2026, the legacy system was configured to only create 4-digit job numbers.
// For years 2027 and beyond, the 3-digit pattern will match no new jobs.
// Once all legacy 3-digit jobs are from prior years, the 3-digit pattern could be removed.
//
// # Implementation
//
// A single SQL query finds the maximum sequence number across both formats:
//   - Uses LIKE with underscore wildcards: "26-___" (3-digit) OR "26-____" (4-digit)
//   - Extracts the numeric suffix and finds MAX using CAST
//
// The underscore patterns naturally exclude sub-jobs like "26-015-01" since they
// have more characters than either pattern matches.
func generateTopLevelJobNumber(app core.App, t jobType) (string, error) {
	return generateTopLevelJobNumberForYear(app, t, time.Now().Year()%100)
}

// generateTopLevelJobNumberForYear is the internal implementation that accepts the
// two-digit year as a parameter. This enables deterministic testing without mocking time.
func generateTopLevelJobNumberForYear(app core.App, t jobType, yy int) (string, error) {
	var prefix string
	if t == jobTypeProposal {
		prefix = fmt.Sprintf("P%02d-", yy)
	} else {
		prefix = fmt.Sprintf("%02d-", yy)
	}

	// Find the maximum sequence number using a single SQL query.
	// LIKE patterns use underscore (_) to match exactly one character:
	//   - prefix + "___"  matches 3-digit legacy format (e.g., "26-015")
	//   - prefix + "____" matches 4-digit current format (e.g., "26-0015")
	// This naturally excludes sub-jobs like "26-015-01" which have more characters.
	var lastNumber int
	err := app.DB().NewQuery(`
		SELECT COALESCE(MAX(CAST(SUBSTR(number, {:prefixLen} + 1) AS INTEGER)), 0)
		FROM jobs
		WHERE number LIKE {:pat3} OR number LIKE {:pat4}
	`).Bind(dbx.Params{
		"prefixLen": len(prefix),
		"pat3":      prefix + "___",
		"pat4":      prefix + "____",
	}).Row(&lastNumber)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	// Generate next unique number (4-digit format going forward)
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
