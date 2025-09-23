package hooks

import (
	"encoding/json"
	"net/http"
	"tybalt/errs"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// ProcessJob enforces business rules for job creation and updates.
func ProcessJob(app core.App, e *core.RecordRequestEvent) error {
	jobRecord := e.Record

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
