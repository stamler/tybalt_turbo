package routes

import (
	"fmt"
	"net/http"
	"time"

	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func createClosePurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		authRecord := e.Auth

		var httpResponseStatusCode int

		err := app.RunInTransaction(func(txApp core.App) error {
			// Check if user has payables_admin claim
			hasPayablesAdminClaim, err := utilities.HasClaim(txApp, authRecord, "payables_admin")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_checking_claim",
					Message: fmt.Sprintf("error checking payables_admin claim: %v", err),
				}
			}
			if !hasPayablesAdminClaim {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized_closure",
					Message: "you are not authorized to close purchase orders",
				}
			}

			// Fetch existing purchase order
			po, err := txApp.FindRecordById("purchase_orders", id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "po_not_found",
					Message: fmt.Sprintf("error fetching purchase order: %v", err),
				}
			}

			// Check if purchase order is allowed to be closed manually per spec:
			// Normal purchase orders cannot be closed manually.
			if po.GetString("type") == "Normal" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "invalid_po_type",
					Message: "Normal -type purchase orders may be cancelled but not manually closed",
				}
			}

			// For Recurring and Cumulative POs, ensure that there is at least one associated expense that is committed (the committed property has a length greater than 0)
			expenses, err := txApp.FindRecordsByFilter("expenses", "purchase_order = {:poId} && committed != '' && committed != NULL", "", 0, 0, dbx.Params{
				"poId": po.Id,
			})
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_expenses",
					Message: fmt.Sprintf("error fetching expenses: %v", err),
				}
			}
			if len(expenses) == 0 {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "no_expenses",
					Message: "only cumulative or recurring purchase orders with at least one associated expense may be closed manually. Cancel the purchase order instead.",
				}
			}

			// Check if purchase order is Active
			if po.GetString("status") != "Active" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_not_active",
					Message: "only active purchase orders can be closed",
				}
			}

			// Update the purchase order status to Closed
			po.Set("closed", time.Now())
			po.Set("closer", authRecord.Id)
			po.Set("status", "Closed")

			// Save the updated record
			if err := txApp.Save(po); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_saving_purchase_order",
					Message: fmt.Sprintf("error saving purchase order: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			if codeError, ok := err.(*CodeError); ok {
				// return apis.NewApiError(httpResponseStatusCode, "error closing purchase order", codeError)
				// TODO: can we have the OnBeforeApiError and OnAfterApiError events fire here by returning an different type of error?
				// How does this relate to HookError?
				// TODO: This is broken. Because an error isn't actually being returned.
				return e.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// return the updated purchase order from the database
		closedPO, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, closedPO)
	}
}
