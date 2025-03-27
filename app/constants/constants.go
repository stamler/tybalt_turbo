// Package constants centralizes application-wide configuration values and thresholds.
// It contains:
// - Feature flags controlling system behavior (POAutoApprove, LIMIT_NON_PO_AMOUNTS)
// - Validation parameters for business rules (MAX_PURCHASE_ORDER_EXCESS_PERCENT/VALUE)
// - Operational limits (RECURRING_MAX_DAYS, NO_PO_EXPENSE_LIMIT)
//
// These constants are used across multiple packages to ensure consistent application
// of business rules and simplify configuration changes.

package constants

const (
	// The maximum number of days between the start and end dates for a recurring
	// purchase order.
	RECURRING_MAX_DAYS = 400

	// These constants are used to determine whether an expense is within the
	// allowed percentage or value of the total of a purchase order. The lesser
	// of the two limits is used to determine if the expense is valid.
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
