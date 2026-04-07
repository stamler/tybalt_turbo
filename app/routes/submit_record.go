package routes

import (
	"fmt"
	"net/http"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

func createSubmitRecordHandler(app core.App, collectionName string) func(e *core.RequestEvent) error {
	// This route handles the submission of a record.
	// It performs the following actions:
	// 1. Retrieves the authenticated user's ID
	// 2. Runs a database transaction to:
	//    a. Fetch the record by ID.
	//    b. Verify that the authenticated user has the same ID as the record's uid.
	//    c. Verify the record is not yet submitted.
	//		d. Verify the record isn't rejected.
	//    e. Set submitted to true.
	//    f. Save the updated record.
	// 3. Returns a success message if submitted, or an error message if any checks fail.
	// This ensures that only the record's owner can submit it.
	return func(e *core.RequestEvent) error {
		if err := requireExpensesEditing(app, collectionName); err != nil {
			return err
		}

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

			// Verify the caller is the record's owner
			if record.Get("uid") != userId {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized",
					Message: "you are not authorized to submit this record",
				}
			}

			// Check if the record is submitted
			if record.GetBool("submitted") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_submitted",
					Message: "this record is already submitted",
				}
			}

			// Check if the record is rejected
			if !record.GetDateTime("rejected").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_rejected",
					Message: "rejected records cannot be submitted",
				}
			}

			if poErr := validateExpensePurchaseOrderIsActive(txApp, record); poErr != nil {
				httpResponseStatusCode = http.StatusBadRequest
				if poErr.Code == "purchase_order_lookup_error" {
					httpResponseStatusCode = http.StatusInternalServerError
				}
				return poErr
			}

			if collectionName == "expenses" {
				currencyInfo, err := utilities.ResolveCurrencyInfo(txApp, record.GetString("currency"))
				if err != nil {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "invalid_currency",
						Message: "referenced currency not found",
					}
				}

				if !utilities.IsHomeCurrencyInfo(currencyInfo) && record.GetString("payment_type") == "Expense" {
					if record.GetFloat("settled_total") <= 0 {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "settled_total_required",
							Message: "settled total is required before submitting a foreign-currency expense",
						}
					}
					if !utilities.IsSettledTotalWithinTolerance(
						record.GetFloat("total"),
						record.GetFloat("settled_total"),
						currencyInfo,
					) {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "settled_total_out_of_range",
							Message: utilities.SettledTotalToleranceMessage(record.GetFloat("total"), currencyInfo),
						}
					}
				}

				if !utilities.IsHomeCurrencyInfo(currencyInfo) &&
					record.GetString("purchase_order") == "" &&
					record.GetString("payment_type") != "Mileage" &&
					record.GetString("payment_type") != "FuelCard" &&
					record.GetString("payment_type") != "PersonalReimbursement" &&
					record.GetString("payment_type") != "Allowance" &&
					utilities.GetNoPOExpenseLimit(txApp) > 0 &&
					record.GetFloat("settled_total") >= utilities.GetNoPOExpenseLimit(txApp) {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "purchase_order_required",
						Message: fmt.Sprintf("a purchase order is required for expenses of $%0.2f or more", utilities.GetNoPOExpenseLimit(txApp)),
					}
				}
			}

			// Set submitted to true
			record.Set("submitted", true)

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

		return e.JSON(http.StatusOK, map[string]string{"message": "Record submitted successfully"})
	}
}
