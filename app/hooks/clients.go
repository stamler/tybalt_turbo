package hooks

import (
	"net/http"

	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

// validateClient performs cross-field validation for the client record.
func validateClient(app core.App, clientRecord *core.Record) error {
	leadID := clientRecord.GetString("business_development_lead")
	if leadID == "" {
		return nil
	}

	hasRequiredClaim, err := utilities.HasClaimByUserID(app, leadID, "busdev")
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
					Message: "business development lead must have busdev claim",
				},
			},
		}
	}

	return nil
}

// ProcessClient enforces business rules for client create/update.
func ProcessClient(app core.App, e *core.RecordRequestEvent) error {
	clientRecord := e.Record

	if err := validateClient(app, clientRecord); err != nil {
		return err
	}

	return nil
}
