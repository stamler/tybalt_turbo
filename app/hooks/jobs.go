package hooks

import (
	"encoding/json"
	"net/http"
	"time"

	"tybalt/errs"
	"tybalt/utilities"

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
