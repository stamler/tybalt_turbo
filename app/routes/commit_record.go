package routes

import (
	"fmt"
	"net/http"
	"time"
	"tybalt/constants"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func createCommitRecordHandler(app core.App, collectionName string) func(e *core.RequestEvent) error {
	// This route handles the committing of a record.
	// It performs the following actions:
	// 1. Retrieves the authenticated user's ID
	// 2. Runs a database transaction to:
	//    a. Fetch the record by ID.
	//    b. Verify that the authenticated user has the required permissions to commit the record.
	//    c. Verify the record is both submitted and approved.
	//		d. Verify the record isn't rejected.
	//    e. Set committed to the current timestamp.
	//    f. Set committer to the authenticated user's ID.
	// 		g. If the record has a committed_week_ending property, update it with the
	//       appropriate week_ending date.
	//		h. If the record is in the expenses collection and has a payment_type of
	//       "Mileage", update the total based on the committed mileage during the
	//       annual fiscal period that corresponds to the date of the record.
	//    i. Save the updated record.
	// 3. Returns a success message if committed, or an error message if any checks fail.
	// This ensures that only the record's owner can submit it.
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.RunInTransaction(func(txApp core.App) error {
			record, err := txApp.FindRecordById(collectionName, e.Request.PathValue("id"))
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("error fetching record: %v", err),
				}
			}

			// Verify the caller has the commit claim by querying the user_claims
			// collection for a record with uid that matches the caller's ID and cid
			// who's name in the claims collection is "commit". If the record exists,
			// the caller has the commit claim.
			hasCommitClaim, err := utilities.HasClaim(txApp, authRecord, "commit")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_user_claims",
					Message: fmt.Sprintf("error fetching user claims: %v", err),
				}
			}
			if !hasCommitClaim {
				httpResponseStatusCode = http.StatusForbidden
				return apis.NewApiError(http.StatusForbidden, "you are not authorized to commit this record", map[string]validation.Error{
					"global": validation.NewError(
						"unauthorized",
						"you are not authorized to commit this record",
					),
				})
			}

			// If the record is not in the time_amendments collection, check if the
			// record is submitted and approved. time_amendments records don't have
			// submitted, approved, or rejected properties.
			if collectionName != "time_amendments" {
				if !record.GetBool("submitted") {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "record_not_submitted",
						Message: "this record is not submitted",
					}
				}

				// Check if the record is approved.
				if record.GetDateTime("approved").IsZero() {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "record_not_approved",
						Message: "this record is not approved",
					}
				}

				// Check if the record is rejected
				if !record.GetDateTime("rejected").IsZero() {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "record_rejected",
						Message: "rejected records cannot be committed",
					}
				}
			}

			// Set commit properties
			now := time.Now()
			record.Set("committer", userId)
			record.Set("committed", now)

			// if the record is an expense, set the committed_week_ending property to
			// the week_ending date that corresponds to the committed timestamp. If
			// the payment_type is "Mileage", also update the total based on the
			// committed mileage during the annual fiscal period that corresponds to
			// the date of the record.
			if record.Collection().Name == "expenses" {
				weekEnding, err := utilities.GenerateWeekEnding(now.Format(time.DateOnly))
				if err != nil {
					return &CodeError{
						Code:    "error_generating_week_ending",
						Message: fmt.Sprintf("error generating week ending: %v", err),
					}
				}
				record.Set("committed_week_ending", weekEnding)

				expenseRateRecord, err := utilities.GetExpenseRateRecord(app, record)
				if err != nil {
					return err
				}

				if record.GetString("payment_type") == "Mileage" {
					totalMileageExpense, mileageErr := utilities.CalculateMileageTotal(app, record, expenseRateRecord)
					if mileageErr != nil {
						return mileageErr
					}
					record.Set("total", totalMileageExpense)
				}

				// Set the `status` property of a referenced purchase_orders record to
				// "Closed" if an expense with a purchase order is committed and
				// necessary conditions are met.
				purchaseOrderId := record.GetString("purchase_order")
				if purchaseOrderId != "" {
					purchaseOrderRecord, err := txApp.FindRecordById("purchase_orders", purchaseOrderId)
					if err != nil {
						// TODO: Verify this error is thrown if the PO is not found. This is
						// necessary to ensure that we're not committing an expense with a
						// purchase order that doesn't exist.
						return fmt.Errorf("purchase order referenced by expense not found: %v", err)
					}
					if purchaseOrderRecord.GetString("status") != "Active" {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "purchase_order_not_active",
							Message: "purchase order is not active",
						}
					}

					// The type field will determine what we do here.
					/*
					   - `Normal` type means just set the Status to Closed, returning an
					     error if it isn't currently `Active`

					   - `Recurring` type means we need to check if an expense has been
					     committed for each recurrence of the PO and set the Status to
					     Closed if so, otherwise doing nothing.

					 	 - `Cumulative` type means we need to check if the cumulative total
					     of all committed expenses against the PO's amount plus the
					     current expense match or exceed the PO total and set the Status
					     to Closed if so, otherwise do nothing.
					*/
					purchaseOrderType := purchaseOrderRecord.GetString("type")
					var dirtyPurchaseOrderRecord bool
					switch purchaseOrderType {
					case "Normal":
						purchaseOrderRecord.Set("status", "Closed")
						dirtyPurchaseOrderRecord = true
					case "Recurring":
						exhausted, err := utilities.RecurringPurchaseOrderExhausted(app, purchaseOrderRecord)
						if err != nil {
							return err
						}
						if exhausted {
							purchaseOrderRecord.Set("status", "Closed")
							dirtyPurchaseOrderRecord = true
						}
					case "Cumulative":
						existingExpensesTotal, err := utilities.CumulativeTotalExpensesForPurchaseOrder(app, purchaseOrderRecord, true)
						if err != nil {
							return err
						}
						pendingExpenseTotal := record.GetFloat("total")

						totalLimit := purchaseOrderRecord.GetFloat("total") * (1.0 + constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT) // initialize with percent limit
						if constants.MAX_PURCHASE_ORDER_EXCESS_VALUE < purchaseOrderRecord.GetFloat("total")*constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT {
							totalLimit = purchaseOrderRecord.GetFloat("total") + constants.MAX_PURCHASE_ORDER_EXCESS_VALUE // use value limit instead
						}

						if existingExpensesTotal+pendingExpenseTotal > totalLimit {
							httpResponseStatusCode = http.StatusBadRequest
							return &CodeError{
								Code:    "exceeded_purchase_order_total",
								Message: "the committed expenses total exceeds the total value of the purchase order beyond the allowed surplus",
							}
						} else if existingExpensesTotal+pendingExpenseTotal >= purchaseOrderRecord.GetFloat("total") {
							// Set the status to Closed since the total of all committed
							// expenses plus the pending expense matches or exceeds the
							// purchase order total
							purchaseOrderRecord.Set("status", "Closed")
							dirtyPurchaseOrderRecord = true
						}
					}
					// Save the purchase order record
					if dirtyPurchaseOrderRecord {
						if err := txApp.Save(purchaseOrderRecord); err != nil {
							httpResponseStatusCode = http.StatusInternalServerError
							return &CodeError{
								Code:    "error_saving_purchase_orders_record",
								Message: fmt.Sprintf("error saving purchase orders record: %v", err),
							}
						}
					}
				}

			}

			// Save the updated record
			if err := txApp.Save(record); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_saving_record",
					Message: fmt.Sprintf("error saving record: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			return e.JSON(httpResponseStatusCode, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"message": "Record committed successfully"})
	}
}
