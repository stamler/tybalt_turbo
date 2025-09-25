package hooks

import (
	"net/http"
	"time"

	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

// cleanClient normalizes fields on the client record before validation.
func cleanClient(app core.App, clientRecord *core.Record) error {
	outstandingBalance := clientRecord.GetFloat("outstanding_balance")

	originalRecord := clientRecord.Original()
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
		clientRecord.Set("outstanding_balance_date", time.Now().Format("2006-01-02"))
	} else if hasOriginal {
		clientRecord.Set("outstanding_balance_date", originalRecord.Get("outstanding_balance_date"))
	}

	return nil
}

// validateClient performs cross-field validation for the client record.
func validateClient(app core.App, clientRecord *core.Record) error {
	leadID := clientRecord.GetString("business_development_lead")
	if leadID == "" {
		return nil
	}

	hasRequiredClaim, err := utilities.HasClaimByUserID(app, leadID, "tapr")
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "error validating business development lead claim",
			Data: map[string]errs.CodeError{
				"business_development_lead": {
					Code:    "claim_check_failed",
					Message: "unable to verify business development lead claim",
				},
			},
		}
	}

	if !hasRequiredClaim {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "invalid business development lead",
			Data: map[string]errs.CodeError{
				"business_development_lead": {
					Code:    "missing_claim",
					Message: "business development lead must have tapr claim",
				},
			},
		}
	}

	return nil
}

// ProcessClient enforces business rules for client create/update.
func ProcessClient(app core.App, e *core.RecordRequestEvent) error {
	clientRecord := e.Record

	if err := cleanClient(app, clientRecord); err != nil {
		return err
	}

	if err := validateClient(app, clientRecord); err != nil {
		return err
	}

	return nil
}
