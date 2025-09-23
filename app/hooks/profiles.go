package hooks

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

// ProcessProfile enforces business rules for profile create/update.
func ProcessProfile(app core.App, e *core.RecordRequestEvent) error {
	defaultDivision := e.Record.GetString("default_division")
	if err := ensureActiveDivision(app, defaultDivision, "default_division"); err != nil {
		if ve, ok := err.(validation.Errors); ok {
			return validation.Errors(ve)
		}
		return err
	}

	return nil
}
