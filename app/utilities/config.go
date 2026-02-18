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

// IsExpensesEditingEnabled checks if expense/vendor/PO creation/editing/absorb is allowed.
// Reads from app_config where key="expenses", checks value.create_edit_absorb.
// Defaults to true (fail-open) if config is missing.
func IsExpensesEditingEnabled(app core.App) (bool, error) {
	return GetConfigBool(app, "expenses", "create_edit_absorb", true)
}

// IsNotificationFeatureEnabled checks whether a notification feature/template is enabled.
// Reads from app_config where key="notifications", and uses templateCode as the JSON key.
// Defaults to false (fail-closed) when config is missing.
func IsNotificationFeatureEnabled(app core.App, templateCode string) (bool, error) {
	return GetConfigBool(app, "notifications", templateCode, false)
}

func GetPurchaseOrderSecondStageTimeoutHours(app core.App) float64 {
	const defaultTimeoutHours = 24.0

	config, err := GetConfigValue(app, "purchase_orders")
	if err != nil || config == nil {
		return defaultTimeoutHours
	}

	rawValue, ok := config["second_stage_timeout_hours"]
	if !ok {
		return defaultTimeoutHours
	}

	switch v := rawValue.(type) {
	case float64:
		if v > 0 {
			return v
		}
	case int:
		if v > 0 {
			return float64(v)
		}
	case int64:
		if v > 0 {
			return float64(v)
		}
	}

	return defaultTimeoutHours
}

// ErrExpensesEditingDisabled is returned when expense editing is disabled
var ErrExpensesEditingDisabled = &errs.HookError{
	Status:  http.StatusForbidden,
	Message: "expense editing is currently disabled",
	Data: map[string]errs.CodeError{
		"global": {Code: "expenses_editing_disabled", Message: "expense, purchase order, and vendor creation and editing is disabled during transition"},
	},
}
