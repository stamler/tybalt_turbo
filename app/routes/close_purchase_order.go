package routes

import (
	"fmt"
	"net/http"
	"strings"
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
			// One-Time purchase orders cannot be closed manually.
			if po.GetString("type") == "One-Time" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "invalid_po_type",
					Message: "One-Time purchase orders may be cancelled but not manually closed",
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

			// Check inside the closure transaction: a closed PO cannot accept further
			// expense approval or commitment. Automatic closure has separate rules.
			pendingMessage, err := pendingPurchaseOrderExpensesMessage(txApp, po.Id)
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_expenses",
					Message: fmt.Sprintf("error fetching pending expenses: %v", err),
				}
			}
			if pendingMessage != "" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{Code: "pending_expenses", Message: pendingMessage}
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

// pendingPurchaseOrderExpensesMessage describes work that blocks manual closure.
// Rejection retains submitted and can retain approved, so it must be excluded
// explicitly. Drafts and recalled expenses do not block closure. Do not reuse the
// commit queue filter here: approved expenses also block while awaiting settlement.
func pendingPurchaseOrderExpensesMessage(app core.App, purchaseOrderID string) (string, error) {
	var pending []struct {
		Approver     string `db:"approver"`
		ApproverName string `db:"approver_name"`
		Approved     bool   `db:"approved"`
	}
	// DISTINCT lists each assigned expense approver once per approval state.
	// LEFT JOIN preserves blocking expenses even when a profile is missing.
	err := app.DB().NewQuery(`
		SELECT DISTINCT e.approver,
			TRIM(COALESCE(p.given_name, '') || ' ' || COALESCE(p.surname, '')) AS approver_name,
			(COALESCE(e.approved, '') != '') AS approved
		FROM expenses e
		LEFT JOIN profiles p ON p.uid = e.approver
		WHERE e.purchase_order = {:poId}
			AND e.submitted = 1
			AND COALESCE(e.committed, '') = ''
			AND COALESCE(e.rejected, '') = ''
		ORDER BY approver_name, e.approver
	`).Bind(dbx.Params{"poId": purchaseOrderID}).All(&pending)
	if err != nil {
		return "", err
	}

	var names []string
	var awaitingCommitment, missingName bool
	for _, expense := range pending {
		if expense.Approved {
			awaitingCommitment = true
		} else if strings.TrimSpace(expense.ApproverName) == "" {
			missingName = true
		} else {
			names = append(names, expense.ApproverName)
		}
	}
	if missingName {
		// Missing display data must not remove the block or leave an empty list.
		names = append(names, "an approver whose name is unavailable")
	}
	var messages []string
	if len(names) > 0 {
		messages = append(messages, "This PO has one or more expenses awaiting approval by "+strings.Join(names, ", ")+".")
	}
	if awaitingCommitment {
		messages = append(messages, "This PO has one or more approved expenses awaiting commitment.")
	}
	return strings.Join(messages, " "), nil
}
