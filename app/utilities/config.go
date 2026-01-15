package utilities

import (
	"encoding/json"
	"net/http"

	"tybalt/errs"

	"github.com/pocketbase/pocketbase/core"
)

// GetConfigValue retrieves a config domain record and returns the parsed JSON value.
// Returns nil if the domain key doesn't exist.
func GetConfigValue(app core.App, domainKey string) (map[string]any, error) {
	record, err := app.FindFirstRecordByData("app_config", "key", domainKey)
	if err != nil {
		return nil, nil // Domain not found
	}

	valueStr := record.GetString("value")
	var value map[string]any
	if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
		return nil, err
	}
	return value, nil
}

// GetConfigBool retrieves a boolean from a domain config.
// domainKey is the row key (e.g., "jobs"), property is the JSON property (e.g., "create_edit_absorb")
func GetConfigBool(app core.App, domainKey string, property string, defaultValue bool) (bool, error) {
	config, err := GetConfigValue(app, domainKey)
	if err != nil {
		return defaultValue, err
	}
	if config == nil {
		return defaultValue, nil
	}

	val, ok := config[property]
	if !ok {
		return defaultValue, nil
	}
	boolVal, ok := val.(bool)
	if !ok {
		return defaultValue, nil
	}
	return boolVal, nil
}

// IsJobsEditingEnabled checks if job creation/editing/absorb is allowed.
// Reads from app_config where key="jobs", checks value.create_edit_absorb.
// Defaults to true (fail-open) if config is missing.
func IsJobsEditingEnabled(app core.App) (bool, error) {
	return GetConfigBool(app, "jobs", "create_edit_absorb", true)
}

// ErrJobsEditingDisabled is returned when job editing is disabled
var ErrJobsEditingDisabled = &errs.HookError{
	Status:  http.StatusForbidden,
	Message: "job editing is currently disabled",
	Data: map[string]errs.CodeError{
		"global": {Code: "jobs_editing_disabled", Message: "job creation and editing is disabled during transition"},
	},
}
