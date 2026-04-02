package routes

import (
	"fmt"
	"net/http"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func createUncommitRecordHandler(app core.App, collectionName string) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		switch collectionName {
		case "expenses":
			if err := requireExpensesEditing(app, collectionName); err != nil {
				return err
			}
		case "time_sheets":
			if err := requireTimeEditing(app); err != nil {
				return err
			}
		default:
			return e.Error(http.StatusInternalServerError, "unsupported uncommit collection", nil)
		}

		authRecord := e.Auth
		if authRecord == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		var httpResponseStatusCode int

		err := app.RunInTransaction(func(txApp core.App) error {
			hasAdminClaim, err := utilities.HasClaim(txApp, authRecord, "admin")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_user_claims",
					Message: fmt.Sprintf("error fetching user claims: %v", err),
				}
			}
			if !hasAdminClaim {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized",
					Message: "you are not authorized to uncommit this record",
				}
			}

			record, err := txApp.FindRecordById(collectionName, e.Request.PathValue("id"))
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("error fetching record: %v", err),
				}
			}

			if record.GetDateTime("committed").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_not_committed",
					Message: "this record is not committed",
				}
			}

			record.Set("committed", "")
			record.Set("committer", "")

			if record.Collection().Fields.GetByName("committed_week_ending") != nil {
				record.Set("committed_week_ending", "")
			}

			if collectionName == "expenses" {
				record.Set("pay_period_ending", "")
			}

			if err := txApp.Save(record); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_saving_record",
					Message: fmt.Sprintf("error saving record: %v", err),
				}
			}

			if collectionName == "expenses" {
				if err := maybeReopenPurchaseOrderAfterExpenseUncommit(txApp, record); err != nil {
					httpResponseStatusCode = http.StatusInternalServerError
					return &CodeError{
						Code:    "error_recalculating_purchase_order",
						Message: fmt.Sprintf("error recalculating purchase order: %v", err),
					}
				}
			}

			return nil
		})

		if err != nil {
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]any{
					"error": codeError.Message,
					"code":  codeError.Code,
				})
			}
			return e.JSON(httpResponseStatusCode, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"message": "Record uncommitted successfully"})
	}
}

func maybeReopenPurchaseOrderAfterExpenseUncommit(app core.App, expenseRecord *core.Record) error {
	purchaseOrderID := expenseRecord.GetString("purchase_order")
	if purchaseOrderID == "" {
		return nil
	}

	purchaseOrderRecord, err := app.FindRecordById("purchase_orders", purchaseOrderID)
	if err != nil {
		return fmt.Errorf("purchase order referenced by expense not found: %w", err)
	}

	if purchaseOrderRecord.GetString("status") != "Closed" {
		return nil
	}

	shouldReopen := false
	switch purchaseOrderRecord.GetString("type") {
	case "One-Time":
		var countResult struct {
			Count int `db:"count"`
		}
		if err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM expenses
			WHERE purchase_order = {:purchase_order}
			  AND committed != ''
		`).Bind(dbx.Params{
			"purchase_order": purchaseOrderID,
		}).One(&countResult); err != nil {
			return err
		}
		shouldReopen = countResult.Count == 0
	case "Recurring":
		exhausted, err := utilities.RecurringPurchaseOrderExhausted(app, purchaseOrderRecord)
		if err != nil {
			return err
		}
		shouldReopen = !exhausted
	case "Cumulative":
		total, err := utilities.CumulativeTotalExpensesForPurchaseOrder(app, purchaseOrderRecord, true)
		if err != nil {
			return err
		}
		shouldReopen = total < purchaseOrderRecord.GetFloat("total")
	}

	if !shouldReopen {
		return nil
	}

	purchaseOrderRecord.Set("status", "Active")
	purchaseOrderRecord.Set("closed", "")
	purchaseOrderRecord.Set("closer", "")
	purchaseOrderRecord.Set("closed_by_system", false)

	return app.Save(purchaseOrderRecord)
}
