package hooks

import (
	"time"

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
	// TODO: if the business_development_lead must have a particular claim, include that validation here
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
