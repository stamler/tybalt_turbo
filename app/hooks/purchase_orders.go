// This file implements cleaning and validation rules for the purchase_orders
// collection.

package hooks

import (
	"errors"
	"fmt"
	"net/http"
	"time"
	"tybalt/constants"
	"tybalt/errs"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The cleanPurchaseOrder function is used to remove properties from the
// purchase_order record that are not allowed to be set based on the value of
// the record's type property. It is also used to set the approval_total field,
// which matches the total field unless the type is Recurring. It is called by
// ProcessPurchaseOrder to reduce the number of fields that need to be
// validated.
func cleanPurchaseOrder(app core.App, purchaseOrderRecord *core.Record) error {
	// initialize approval_total to total. This will be changed if the PO is
	// recurring.
	purchaseOrderRecord.Set("approval_total", purchaseOrderRecord.GetFloat("total"))

	typeString := purchaseOrderRecord.GetString("type")

	// Normal and Cumulative Purchase both have empty values for
	// end_date and frequency
	if typeString == "Normal" || typeString == "Cumulative" {
		purchaseOrderRecord.Set("end_date", "")
		purchaseOrderRecord.Set("frequency", "")
	}

	if typeString == "Recurring" {
		_, calculatedTotal, err := utilities.CalculateRecurringPurchaseOrderTotalValue(app, purchaseOrderRecord)
		if err != nil {
			var hookErr *errs.HookError
			if errors.As(err, &hookErr) {
				return err
			} else {
				return &errs.HookError{
					Status:  http.StatusInternalServerError,
					Message: "hook error when calculating recurring PO total value",
					Data: map[string]errs.CodeError{
						"global": {
							Code:    "error_calculating_total_value",
							Message: fmt.Sprintf("error calculating recurring PO total value: %v", err),
						},
					},
				}
			}
		}
		purchaseOrderRecord.Set("approval_total", calculatedTotal)
	}

	// Clear priority_second_approver if approval_total <= the lowest threshold
	thresholds, err := utilities.GetPOApprovalThresholds(app)
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "hook error when fetching po approval thresholds",
		}
	}
	if purchaseOrderRecord.GetFloat("approval_total") <= thresholds[0] {
		purchaseOrderRecord.Set("priority_second_approver", "")
	}

	// Clear all rejection fields here. ProcessPurchaseOrder, which calls
	// cleanPurchaseOrder, is only ever called when a user is creating or
	// updating a PO. POs cannot be rejected upon creation, so clearing rejection
	// fields is idempotent. Upon update, rejection fields should be cleared if
	// any changes are made to the record, on the assumption that the user is
	// preparing to resubmit the PO after making changes.
	purchaseOrderRecord.Set("rejected", "")
	purchaseOrderRecord.Set("rejection_reason", "")
	purchaseOrderRecord.Set("rejector", "")

	return nil
}

// cross-field validation is performed in this function. It is expected that the
// purchase_order record has already been cleaned by the cleanPurchaseOrder
// function. This ensures that only the fields that are allowed to be set are
// present in the record prior to validation. The function returns an error if
// the record is invalid, otherwise it returns nil.
func validatePurchaseOrder(app core.App, purchaseOrderRecord *core.Record) error {
	isRecurring := purchaseOrderRecord.GetString("type") == "Recurring"
	isChild := purchaseOrderRecord.GetString("parent_po") != ""

	if isChild {
		// Validate parent PO is active and cumulative
		parentPO, err := app.FindRecordById("purchase_orders", purchaseOrderRecord.GetString("parent_po"))
		if err != nil {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when fetching parent PO",
				Data: map[string]errs.CodeError{
					"parent_po": {
						Code:    "not_found",
						Message: "parent PO not found",
					},
				},
			}
		}

		// Validate parent PO is not itself a child
		if parentPO.GetString("parent_po") != "" {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when validating parent PO",
				Data: map[string]errs.CodeError{
					"parent_po": {
						Code:    "child_po_cannot_be_parent",
						Message: "parent PO cannot be itself a child",
					},
				},
			}
		}

		if parentPO.GetString("status") != "Active" {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when validating parent PO",
				Data: map[string]errs.CodeError{
					"parent_po": {
						Code:    "invalid_status",
						Message: "parent PO must be active",
					},
				},
			}
		}

		if parentPO.GetString("type") != "Cumulative" {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when validating parent PO",
				Data: map[string]errs.CodeError{
					"parent_po": {
						Code:    "invalid_type",
						Message: "parent PO must be cumulative",
					},
				},
			}
		}

		// Validate that other child POs with status "Unapproved" do not exist
		otherChildPOs, err := app.FindRecordsByFilter("purchase_orders", "parent_po = {:parentId} && status != 'Closed' && status != 'Cancelled'", "", 0, 0, dbx.Params{
			"parentId": parentPO.Id,
		})
		if err != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "hook error when fetching other child POs",
				Data: map[string]errs.CodeError{
					"parent_po": {
						Code:    "internal_server_error",
						Message: "error searching for other child POs",
					},
				},
			}
		}

		if len(otherChildPOs) > 0 {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when validating parent PO",
				Data: map[string]errs.CodeError{
					"parent_po": {
						Code:    "existing_children_with_blocking_status",
						Message: "other child POs that are not 'Closed' or 'Cancelled' already exist",
					},
				},
			}
		}

		// Validate fields match parent PO
		fieldsToMatch := []string{"job", "payment_type", "category", "description", "vendor"}
		for _, field := range fieldsToMatch {
			if purchaseOrderRecord.GetString(field) != parentPO.GetString(field) {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "hook error when validating parent PO",
					Data: map[string]errs.CodeError{
						field: {
							Code:    "value_mismatch",
							Message: fmt.Sprintf("field %s must match parent PO's %s", field, field),
						},
					},
				}
			}
		}
	}

	dateAsTime, parseErr := time.Parse("2006-01-02", purchaseOrderRecord.Get("date").(string))
	if parseErr != nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when validating date",
			Data: map[string]errs.CodeError{
				"date": {
					Code:    "invalid_date",
					Message: "date must be a valid date",
				},
			},
		}
	}

	// Validate priority_second_approver if set
	prioritySecondApproverIsAuthorized := func(app core.App, purchaseOrderRecord *core.Record) validation.RuleFunc {
		return func(value any) error {
			prioritySecondApproverId := purchaseOrderRecord.GetString("priority_second_approver")
			if prioritySecondApproverId != "" {
				division := purchaseOrderRecord.GetString("division")

				// Get list of eligible second approvers
				approvers, _, err := utilities.GetPOApprovers(app, nil, division, purchaseOrderRecord.GetFloat("approval_total"), true)
				if err != nil {
					return &errs.HookError{
						Status:  http.StatusInternalServerError,
						Message: "hook error when checking eligible second approvers",
						Data: map[string]errs.CodeError{
							"global": {
								Code:    "error_checking_approvers",
								Message: fmt.Sprintf("error checking eligible second approvers: %v", err),
							},
						},
					}
				}

				// Check if prioritySecondApprover is in the list of eligible approvers
				valid := false
				for _, approver := range approvers {
					if approver.ID == prioritySecondApproverId {
						valid = true
						break
					}
				}

				if !valid {
					return validation.NewError("invalid_priority_second_approver", "The selected priority second approver is not authorized to approve this purchase order")
				}
			}
			return nil
		}
	}

	validationsErrors := validation.Errors{
		"date": validation.Validate(
			purchaseOrderRecord.Get("date"),
			validation.Required.Error("date is required"),
			validation.Date("2006-01-02").Error("must be a valid date"),
		),
		"end_date": validation.Validate(
			purchaseOrderRecord.Get("end_date"),
			validation.When(isRecurring,
				validation.Required.Error("end_date is required for recurring purchase orders"),
				validation.Date("2006-01-02").Error("must be a valid date").Min(dateAsTime).RangeError("end date must be after start date").Max(dateAsTime.AddDate(0, 0, constants.RECURRING_MAX_DAYS)).RangeError(fmt.Sprintf("end date must be within %v days of the start date", constants.RECURRING_MAX_DAYS)),
			).Else(
				validation.In("").Error("end_date is not permitted for non-recurring purchase orders"),
			),
		),
		"priority_second_approver": validation.Validate(purchaseOrderRecord.Get("priority_second_approver"), validation.By(prioritySecondApproverIsAuthorized(app, purchaseOrderRecord))),
		"frequency": validation.Validate(
			purchaseOrderRecord.Get("frequency"),
			validation.When(isRecurring,
				validation.Required.Error("frequency is required for recurring purchase orders"),
			).Else(
				validation.In("").Error("frequency is not permitted for non-recurring purchase orders"))),
		"description": validation.Validate(purchaseOrderRecord.Get("description"), validation.Length(5, 0).Error("must be at least 5 characters")),
		"approver":    validation.Validate(purchaseOrderRecord.GetString("approver"), validation.By(utilities.PoApproverPropsHasDivisionPermission(app, constants.PO_APPROVER_CLAIM_ID, purchaseOrderRecord.GetString("division")))),
		"total":       validation.Validate(purchaseOrderRecord.GetFloat("total"), validation.Max(constants.MAX_APPROVAL_TOTAL)),
		"type":        validation.Validate(purchaseOrderRecord.GetString("type"), validation.When(isChild, validation.In("Normal").Error("child POs must be of type Normal"))),
	}.Filter()

	return validationsErrors
}

// The ProcessPurchaseOrder function is used to validate the purchase_order
// record before it is created or updated. A lot of the work is done by
// PocketBase itself so this is for cross-field validation. If the
// purchase_order record is invalid this function throws an error explaining
// which field(s) are invalid and why.
func ProcessPurchaseOrder(app core.App, e *core.RecordRequestEvent) error {
	record := e.Record
	// get the auth record from the context
	authRecord := e.Auth

	// If the uid property is not equal to the authenticated user's uid, return an
	// error.
	if record.GetString("uid") != authRecord.Id {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when validating uid",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "value_mismatch",
					Message: "uid must be equal to the authenticated user's id",
				},
			},
		}
	}

	// set properties to nil if they are not allowed to be set based on the type
	cleanErr := cleanPurchaseOrder(app, record)
	if cleanErr != nil {
		return cleanErr
	}

	// validate the purchase_order record
	if validationErr := validatePurchaseOrder(app, record); validationErr != nil {
		return validationErr
	}

	// If this is a new record, send a notification to the approver
	if record.IsNew() {

		// Generate a new id for the record here so that the notification can
		// reference it
		// https://github.com/pocketbase/pocketbase/discussions/6170
		// https://pocketbase.io/docs/collections/#textfield
		record.Set("id:autogenerate", "")

		notificationCollection, err := app.FindCollectionByNameOrId("notifications")
		if err != nil {
			return err
		}

		notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
			"code": "po_approval_required",
		})
		if err != nil {
			return err
		}

		notificationRecord := core.NewRecord(notificationCollection)
		notificationRecord.Set("recipient", record.GetString("approver"))
		notificationRecord.Set("template", notificationTemplate.Id)
		notificationRecord.Set("status", "pending")
		notificationRecord.Set("user", authRecord.Id)
		notificationRecord.Set("data", map[string]any{
			"POId": record.Id,
		})

		if err := app.Save(notificationRecord); err != nil {
			return err
		}
	}

	return nil
}
