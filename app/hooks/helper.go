package hooks

import (
	"net/http"
	"tybalt/errs"

	"github.com/pocketbase/pocketbase/core"
)

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
