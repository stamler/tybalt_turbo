package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func createRejectExpenseHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	// This route handles the rejection of an expense.
	// It performs the following actions:
	// 1. Gets the expense ID from the URL.
	// 2. Validates the request body for a valid rejection reason.
	// 3. Retrieves the authenticated user's ID.
	// 4. Runs a database transaction to:
	//    a. Fetch the expense by ID.
	//    b. Verify that the authenticated user is the assigned approver.
	//    c. Check if the expense is submitted and not committed or already rejected.
	//    d. Set the rejection timestamp, reason, and rejector.
	//    e. Save the updated expense.
	// 5. Returns a success message if rejected, or an error message if any checks fail.
	// This ensures that only valid, submitted expenses can be rejected by the correct user.
	return func(c echo.Context) error {

		id := c.PathParam("id")

		var req RejectionRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "you must provide a rejection reason",
				"code":    "invalid_request_body",
			})
		}

		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			expense, err := txDao.FindRecordById("expenses", id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("error fetching expense: %v", err),
				}
			}

			// Check if the user is the approver
			if expense.GetString("approver") != userId {
				httpResponseStatusCode = http.StatusUnauthorized
				return &CodeError{
					Code:    "rejection_unauthorized",
					Message: "you are not authorized to reject this expense",
				}
			}

			// Check if the expense is submitted
			if !expense.GetBool("submitted") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "expense_not_submitted",
					Message: "only submitted expenses can be rejected",
				}
			}

			// Check if the expense is committed
			if !expense.GetDateTime("committed").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "expense_committed",
					Message: "committed expenses cannot be rejected",
				}
			}

			// Check if the expense is already rejected
			if expense.GetDateTime("rejected").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "expense_already_rejected",
					Message: "this expense is already rejected",
				}
			}

			// Check if the rejection reason is at least 4 characters long
			if len(req.RejectionReason) < 4 {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "rejection_reason_too_short",
					Message: "rejection reason must be at least 4 characters long",
				}
			}

			// Set the rejection timestamp, reason, and rejector
			expense.Set("rejected", time.Now())
			expense.Set("rejection_reason", req.RejectionReason)
			expense.Set("rejector", userId)

			// Save the updated expense
			if err := txDao.SaveRecord(expense); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "expense_save_error",
					Message: fmt.Sprintf("error saving expense: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			if codeError, ok := err.(*CodeError); ok {
				return c.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Expense rejected successfully"})
	}
}
