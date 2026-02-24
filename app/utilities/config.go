package utilities

import (
	"encoding/json"
	"fmt"
	"net/http"

	"tybalt/constants"
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

// POExpenseExcessConfig holds the configuration for how much expenses can
// exceed a purchase order total.
type POExpenseExcessConfig struct {
	Percent float64 // fractional, e.g. 0.05 = 5%
	Value   float64 // absolute dollar amount
	Mode    string  // "lesser_of" or "greater_of"
}

// POExpenseLimitResult holds the computed total limit and a human-readable
// description of the excess amount for use in error messages.
type POExpenseLimitResult struct {
	TotalLimit float64
	ExcessText string // e.g. "5.00%" or "$100.00"
}

// GetPOExpenseExcessConfig reads the po_expense_allowed_excess config from the
// "expenses" domain in app_config. Returns defaults matching the existing
// hardcoded constants when the config is missing or invalid.
func GetPOExpenseExcessConfig(app core.App) POExpenseExcessConfig {
	defaults := POExpenseExcessConfig{
		Percent: constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT,
		Value:   constants.MAX_PURCHASE_ORDER_EXCESS_VALUE,
		Mode:    "lesser_of",
	}

	config, err := GetConfigValue(app, "expenses")
	if err != nil || config == nil {
		return defaults
	}

	rawExcess, ok := config["po_expense_allowed_excess"]
	if !ok {
		return defaults
	}

	excessMap, ok := rawExcess.(map[string]any)
	if !ok {
		return defaults
	}

	result := defaults // start from defaults, override only valid fields

	// "percent" is stored as a human-readable percentage (e.g. 5 means 5%)
	// and normalized to a fraction for internal use.
	if percent, err := coerceFloat64(excessMap["percent"]); err == nil && percent >= 0 && percent <= 100 {
		result.Percent = percent / 100
	}

	if value, err := coerceFloat64(excessMap["value"]); err == nil && value >= 0 {
		result.Value = value
	}

	if mode, ok := excessMap["mode"].(string); ok {
		if mode == "lesser_of" || mode == "greater_of" {
			result.Mode = mode
		}
	}

	return result
}

// CalculatePOExpenseTotalLimit computes the maximum allowed expense total for a
// purchase order, based on the excess configuration. The ExcessText field
// describes which limit was applied (percent or absolute value).
//
// Note: when poTotal is 0 the percent excess is also 0, so the percent branch
// is always selected and ExcessText will report a percentage even though the
// effective allowed excess is $0. This is acceptable because $0 POs are not
// expected in practice.
func CalculatePOExpenseTotalLimit(poTotal float64, cfg POExpenseExcessConfig) POExpenseLimitResult {
	percentLimit := poTotal * (1.0 + cfg.Percent)
	valueLimit := poTotal + cfg.Value

	var totalLimit float64
	var excessText string

	switch cfg.Mode {
	case "greater_of":
		if cfg.Value >= poTotal*cfg.Percent {
			totalLimit = valueLimit
			excessText = fmt.Sprintf("$%0.2f", cfg.Value)
		} else {
			totalLimit = percentLimit
			excessText = fmt.Sprintf("%0.2f%%", cfg.Percent*100)
		}
	default: // "lesser_of"
		if cfg.Value < poTotal*cfg.Percent {
			totalLimit = valueLimit
			excessText = fmt.Sprintf("$%0.2f", cfg.Value)
		} else {
			totalLimit = percentLimit
			excessText = fmt.Sprintf("%0.2f%%", cfg.Percent*100)
		}
	}

	return POExpenseLimitResult{
		TotalLimit: totalLimit,
		ExcessText: excessText,
	}
}

// coerceFloat64 converts a JSON numeric value (which may be float64, int, or
// int64 after json.Unmarshal) to float64.
func coerceFloat64(v any) (float64, error) {
	if v == nil {
		return 0, fmt.Errorf("nil value")
	}
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}

// GetNoPOExpenseLimit reads the no-PO expense limit from the "expenses" domain
// in app_config. Returns the default from constants when the config is missing,
// the key is absent, or the value is negative.
// A value of 0 means all non-exempt expenses require a PO.
func GetNoPOExpenseLimit(app core.App) float64 {
	config, err := GetConfigValue(app, "expenses")
	if err != nil || config == nil {
		return constants.NO_PO_EXPENSE_LIMIT
	}
	if limit, err := coerceFloat64(config["no_po_expense_limit"]); err == nil && limit >= 0 {
		return limit
	}
	return constants.NO_PO_EXPENSE_LIMIT
}

// ErrExpensesEditingDisabled is returned when expense editing is disabled
var ErrExpensesEditingDisabled = &errs.HookError{
	Status:  http.StatusForbidden,
	Message: "expense editing is currently disabled",
	Data: map[string]errs.CodeError{
		"global": {Code: "expenses_editing_disabled", Message: "expense, purchase order, and vendor creation and editing is disabled during transition"},
	},
}
