package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
