package routes

import (
	"fmt"
	"net/http"

	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/apis"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func createClosePurchaseOrderHandler(app core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.PathParam("id")
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			// Check if user has payables_admin claim
			hasPayablesAdminClaim, err := utilities.HasClaim(txDao, userId, "payables_admin")
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
			po, err := txDao.FindRecordById("purchase_orders", id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "po_not_found",
					Message: fmt.Sprintf("error fetching purchase order: %v", err),
				}
			}

			// Check if purchase order is of type Cumulative
			if po.GetString("type") != "Cumulative" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "invalid_po_type",
					Message: "only cumulative purchase orders can be closed manually",
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
			po.Set("status", "Closed")

			// Save the updated record
			if err := txDao.SaveRecord(po); err != nil {
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
				return c.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Purchase order closed successfully"})
	}
}
