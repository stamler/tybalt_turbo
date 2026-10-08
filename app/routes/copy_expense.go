package routes

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"tybalt/errs"
	"tybalt/hooks"

	"github.com/pocketbase/pocketbase/core"
)

// copyExpenseFields lists the user-entered expense fields carried onto a copy.
// Everything else (workflow state, approver, totals, attachments, settlement,
// pay period) is either left blank or recomputed by hooks.ProcessExpense for
// the new date.
var copyExpenseFields = []string{
	"uid",
	"payment_type",
	"allowance_types",
	"distance",
	"description",
	"job",
	"division",
	"branch",
	"category",
	"kind",
	"currency",
}

// createCopyExpenseHandler returns a route handler that duplicates the given
// expenses record with the date moved forward by one day. Only Allowance and
// Mileage expenses are eligible because their totals and descriptions are
// derived server-side from the date and neither requires a receipt. The
// handler enforces:
//  1. The authenticated user must own the original record (uid matches).
//  2. The payment_type must be Allowance or Mileage.
//  3. The original must not be linked to a purchase order, since a copy would
//     draw down the PO a second time.
//
// The source may be in any workflow state; it is never modified. The copy is a
// new unsubmitted draft run through the normal expense processing and
// validation for its new date, so e.g. current rates and insurance expiry
// apply. On success the handler returns a 201 JSON payload containing the id
// of the new record.
func createCopyExpenseHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireExpensesEditing(app, "expenses"); err != nil {
			return err
		}

		authRecord := e.Auth

		var httpResponseStatusCode int
		var newRecordId string

		err := app.RunInTransaction(func(txApp core.App) error {
			originalId := e.Request.PathValue("id")

			original, err := txApp.FindRecordById("expenses", originalId)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("expense %s not found", originalId),
				}
			}

			if original.GetString("uid") != authRecord.Id {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized",
					Message: "you are not the owner of this expense",
				}
			}

			switch original.GetString("payment_type") {
			case "Allowance", "Mileage":
			default:
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "unsupported_payment_type",
					Message: "only Allowance and Mileage expenses can be copied to tomorrow",
				}
			}

			if original.GetString("purchase_order") != "" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "linked_to_purchase_order",
					Message: "cannot copy an expense that is linked to a purchase order",
				}
			}

			originalDate, parseErr := time.Parse("2006-01-02", original.GetString("date"))
			if parseErr != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "invalid_date",
					Message: fmt.Sprintf("invalid date format on original record: %v", parseErr),
				}
			}

			newRecord := core.NewRecord(original.Collection())
			for _, field := range copyExpenseFields {
				newRecord.Set(field, original.Get(field))
			}
			newRecord.Set("date", originalDate.AddDate(0, 0, 1).Format("2006-01-02"))

			// ProcessExpense reads the request body for attachment and
			// source_expense inputs. Give it a body-less clone of this request so
			// nothing the client sends to the copy endpoint can influence the copy.
			copyRequest := e.Request.Clone(e.Request.Context())
			copyRequest.Body = http.NoBody
			copyRequest.ContentLength = 0
			copyRequest.Header.Del("Content-Type")
			copyEvent := &core.RecordRequestEvent{
				RequestEvent: &core.RequestEvent{App: txApp, Auth: authRecord},
				Record:       newRecord,
			}
			copyEvent.Request = copyRequest
			copyEvent.Response = e.Response

			if err := hooks.ProcessExpense(txApp, copyEvent); err != nil {
				return err
			}

			if err := txApp.Save(newRecord); err != nil {
				return err
			}

			newRecordId = newRecord.Id
			return nil
		})

		if err != nil {
			if codeErr, ok := err.(*CodeError); ok {
				if httpResponseStatusCode == 0 {
					httpResponseStatusCode = http.StatusBadRequest
				}
				return e.JSON(httpResponseStatusCode, map[string]any{
					"error": codeErr.Message,
					"code":  codeErr.Code,
				})
			}
			var hookErr *errs.HookError
			if errors.As(err, &hookErr) {
				return e.JSON(hookErr.Status, hookErr)
			}
			return expenseWriteError(e, err)
		}

		return e.JSON(http.StatusCreated, map[string]string{
			"message":       "Expense copied to tomorrow",
			"new_record_id": newRecordId,
		})
	}
}
