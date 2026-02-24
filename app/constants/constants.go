// Package constants centralizes application-wide configuration values and thresholds.
// It contains:
// - Feature flags controlling system behavior (POAutoApprove, LIMIT_NON_PO_AMOUNTS)
// - Validation parameters for business rules (MAX_PURCHASE_ORDER_EXCESS_PERCENT/VALUE)
// - Operational limits (RECURRING_MAX_DAYS, NO_PO_EXPENSE_LIMIT)
// - Validation patterns (LocationPlusCodeRegex)
//
// These constants are used across multiple packages to ensure consistent application
// of business rules and simplify configuration changes.

package constants

import (
	"regexp"
	"time"
)

// LocationPlusCodeRegex validates Plus Code location format.
// Plus Codes are 8 characters followed by '+' and 2-3 characters.
// Valid characters are: 23456789CFGHJMPQRVWX
var LocationPlusCodeRegex = regexp.MustCompile(`^[23456789CFGHJMPQRVWX]{8}\+[23456789CFGHJMPQRVWX]{2,3}$`)

const (
	// admin_profiles defaults
	DEFAULT_WORK_WEEK_HOURS = 40
	DEFAULT_CHARGE_OUT_RATE = 50

	// The default branch id for 'ThunderBay' from the branches collection in the
	// test database.
	DEFAULT_BRANCH_ID = "80875lm27v8wgi4"

	// The maximum number of days between the start and end dates for a recurring
	// purchase order.
	RECURRING_MAX_DAYS = 400

	// Default values for the allowed excess on purchase order expenses.
	// These can be overridden via the "expenses" domain in app_config under
	// the "po_expense_allowed_excess" property.
	MAX_PURCHASE_ORDER_EXCESS_PERCENT = 0.05
	MAX_PURCHASE_ORDER_EXCESS_VALUE   = 100.0

	// This feature flag is used to limit the total of expenses that don't have a
	// corresponding purchase order.
	LIMIT_NON_PO_AMOUNTS = true
	NO_PO_EXPENSE_LIMIT  = 100.0

	// Maximum approval_total for a purchase_orders record that can be approved
	// by a second approver.
	MAX_APPROVAL_TOTAL = 1000000.0

	// The claim ID for the po_approver claim
	PO_APPROVER_CLAIM_ID = "5vh881k048bboim"
)

// The epoch date for the payroll, initialized as a variable.
var PAYROLL_EPOCH = time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
