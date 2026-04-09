package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

// validateExpensePurchaseOrderIsActive ensures that a referenced purchase order
// exists and is Active before allowing submit/approve transitions for expenses.
func validateExpensePurchaseOrderIsActive(app core.App, record *core.Record) *CodeError {
	if record.Collection().Name != "expenses" {
		return nil
	}

	purchaseOrderID := record.GetString("purchase_order")
	if purchaseOrderID == "" {
		return nil
	}

	purchaseOrderRecord, err := app.FindRecordById("purchase_orders", purchaseOrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &CodeError{
				Code:    "purchase_order_not_found",
				Message: "purchase order referenced by expense not found",
			}
		}
		return &CodeError{
			Code:    "purchase_order_lookup_error",
			Message: fmt.Sprintf("error fetching purchase order: %v", err),
		}
	}

	if purchaseOrderRecord.GetString("status") != "Active" {
		return &CodeError{
			Code:    "purchase_order_not_active",
			Message: "purchase order is not active",
		}
	}

	return nil
}

// expensePurchaseOrderOwnerUIDMismatch compares the expense owner to the linked
// purchase order owner. We only flag a mismatch when both IDs are present.
func expensePurchaseOrderOwnerUIDMismatch(expenseUID string, purchaseOrderOwnerUID string) bool {
	expenseUID = strings.TrimSpace(expenseUID)
	purchaseOrderOwnerUID = strings.TrimSpace(purchaseOrderOwnerUID)

	return expenseUID != "" && purchaseOrderOwnerUID != "" && expenseUID != purchaseOrderOwnerUID
}

// validateExpenseNoPurchaseOrderLimit enforces the configured CAD-denominated
// no-PO cap for foreign-currency expenses at route-time transitions that depend
// on settled_total values.
func validateExpenseNoPurchaseOrderLimit(app core.App, record *core.Record, currencyInfo utilities.CurrencyInfo, settledTotal float64) *CodeError {
	if record.Collection().Name != "expenses" {
		return nil
	}
	if utilities.IsHomeCurrencyInfo(currencyInfo) {
		return nil
	}
	if strings.TrimSpace(record.GetString("purchase_order")) != "" {
		return nil
	}

	switch record.GetString("payment_type") {
	case "Mileage", "FuelCard", "PersonalReimbursement", "Allowance":
		return nil
	}

	limit := utilities.GetNoPOExpenseLimit(app)
	if limit <= 0 {
		return nil
	}
	if settledTotal < limit {
		return nil
	}

	return &CodeError{
		Code:    "purchase_order_required",
		Message: fmt.Sprintf("a purchase order is required for expenses of $%0.2f or more", limit),
	}
}

func validatePositiveForeignCurrencyRate(currencyInfo utilities.CurrencyInfo) *CodeError {
	if err := utilities.RequirePositiveForeignCurrencyRate(currencyInfo); err != nil {
		return &CodeError{
			Code:    "missing_rate",
			Message: "selected foreign currency is missing an exchange rate",
		}
	}

	return nil
}
