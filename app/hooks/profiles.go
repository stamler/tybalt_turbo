package hooks

import (
	"tybalt/utilities"

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

	// Validate that manager (and alternate_manager if provided) have the `tapr` claim.
	// UI restricts choices to the `managers` view (users with `tapr`), but we
	// enforce it here for correctness.
	managerUID := e.Record.GetString("manager")
	if managerUID != "" {
		hasTapr, err := utilities.HasClaimByUserID(app, managerUID, "tapr")
		if err != nil {
			return err
		}
		if !hasTapr {
			return validation.Errors{
				"manager": validation.NewError("invalid_manager", "manager must have tapr claim"),
			}
		}
	}

	alternateManagerUID := e.Record.GetString("alternate_manager")
	if alternateManagerUID != "" {
		hasTapr, err := utilities.HasClaimByUserID(app, alternateManagerUID, "tapr")
		if err != nil {
			return err
		}
		if !hasTapr {
			return validation.Errors{
				"alternate_manager": validation.NewError("invalid_alternate_manager", "alternate manager must have tapr claim"),
			}
		}
	}

	return nil
}
