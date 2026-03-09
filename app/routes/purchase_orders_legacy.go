package routes

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"tybalt/constants"
	"tybalt/errs"
	"tybalt/hooks"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

var legacyPurchaseOrderNumberPattern = regexp.MustCompile(`^(25|26)(0[1-9]|1[0-2])-5\d{3}$`)

var legacyPurchaseOrderEditableFields = map[string]struct{}{
	"uid":          {},
	"approver":     {},
	"po_number":    {},
	"date":         {},
	"division":     {},
	"description":  {},
	"payment_type": {},
	"total":        {},
	"vendor":       {},
	"type":         {},
	"kind":         {},
	"branch":       {},
	"job":          {},
}

var legacyPurchaseOrderRequiredFields = []struct {
	Name    string
	Message string
}{
	{Name: "uid", Message: "Staff member is required."},
	{Name: "approver", Message: "Approver is required."},
	{Name: "po_number", Message: "Legacy PO number is required."},
	{Name: "date", Message: "Date is required."},
	{Name: "division", Message: "Division is required."},
	{Name: "description", Message: "Description is required."},
	{Name: "payment_type", Message: "Payment type is required."},
	{Name: "vendor", Message: "Vendor is required."},
	{Name: "type", Message: "Type is required."},
	{Name: "kind", Message: "Kind is required."},
	{Name: "branch", Message: "Branch is required."},
}

func requireLegacyPurchaseOrderRouteAccess(app core.App, authRecord *core.Record) error {
	if authRecord == nil || strings.TrimSpace(authRecord.Id) == "" {
		return &errs.HookError{
			Status:  http.StatusUnauthorized,
			Message: "legacy purchase order access requires authentication",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "authentication_required",
					Message: "authentication is required",
				},
			},
		}
	}

	if err := requireExpensesEditing(app, "purchase_orders"); err != nil {
		return &errs.HookError{
			Status:  http.StatusForbidden,
			Message: err.Error(),
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "expenses_editing_disabled",
					Message: err.Error(),
				},
			},
		}
	}

	enabled, err := utilities.IsLegacyPOCreateUpdateEnabled(app)
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "unable to validate legacy purchase order configuration",
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "legacy_purchase_order_config_error",
					Message: "unable to validate legacy purchase order configuration",
				},
			},
		}
	}
	if !enabled {
		return &errs.HookError{
			Status:  http.StatusForbidden,
			Message: "legacy purchase order create/update is disabled",
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "legacy_purchase_order_create_update_disabled",
					Message: "legacy purchase order create/update is currently disabled",
				},
			},
		}
	}

	hasClaim, err := utilities.HasClaim(app, authRecord, constants.LEGACY_PO_CREATE_UPDATE_CLAIM_NAME)
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "unable to validate legacy purchase order permissions",
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "legacy_purchase_order_claim_check_failed",
					Message: "unable to validate legacy purchase order permissions",
				},
			},
		}
	}
	if !hasClaim {
		return &errs.HookError{
			Status:  http.StatusForbidden,
			Message: "legacy purchase order create/update requires a dedicated claim",
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "legacy_purchase_order_claim_required",
					Message: "you do not have permission to create or edit legacy purchase orders",
				},
			},
		}
	}

	return nil
}

func rejectLegacyPurchaseOrderDisallowedFields(payload map[string]any) error {
	for field := range payload {
		if _, ok := legacyPurchaseOrderEditableFields[field]; ok {
			continue
		}
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "legacy purchase order payload contains a field that is not editable",
			Data: map[string]errs.CodeError{
				field: {
					Code:    "field_not_allowed",
					Message: "this field cannot be edited through the legacy purchase order endpoint",
				},
			},
		}
	}

	return nil
}

func applyLegacyPurchaseOrderPayload(record *core.Record, payload map[string]any) {
	for field, value := range payload {
		if _, ok := legacyPurchaseOrderEditableFields[field]; ok {
			record.Set(field, value)
		}
	}
}

func requireLegacyPurchaseOrderFields(record *core.Record) error {
	for _, field := range legacyPurchaseOrderRequiredFields {
		if strings.TrimSpace(record.GetString(field.Name)) == "" {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "legacy purchase order is missing a required field",
				Data: map[string]errs.CodeError{
					field.Name: {
						Code:    "required",
						Message: field.Message,
					},
				},
			}
		}
	}

	if record.GetFloat("total") <= 0 {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "legacy purchase order is missing a required field",
			Data: map[string]errs.CodeError{
				"total": {
					Code:    "required",
					Message: "Total must be greater than 0.",
				},
			},
		}
	}

	return nil
}

func normalizeLegacyPurchaseOrderRecord(record *core.Record) error {
	record.Set("po_number", strings.TrimSpace(record.GetString("po_number")))
	record.Set("legacy_manual_entry", true)
	record.Set("_imported", false)
	record.Set("status", "Active")
	record.Set("priority_second_approver", "")
	record.Set("second_approver", "")
	record.Set("second_approval", "")
	record.Set("rejector", "")
	record.Set("rejected", "")
	record.Set("rejection_reason", "")
	record.Set("cancelled", "")
	record.Set("canceller", "")
	record.Set("closed", "")
	record.Set("closer", "")
	record.Set("closed_by_system", false)
	record.Set("end_date", "")
	record.Set("frequency", "")
	record.Set("parent_po", "")
	record.Set("category", "")
	record.Set("attachment", "")
	record.Set("attachment_hash", "")

	if !legacyPurchaseOrderNumberPattern.MatchString(record.GetString("po_number")) {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "legacy purchase order number is invalid",
			Data: map[string]errs.CodeError{
				"po_number": {
					Code:    "invalid_legacy_po_number",
					Message: "Legacy PO number must match YYMM-NNNN with YY 25/26 and NNNN in the 5XXX range.",
				},
			},
		}
	}

	switch record.GetString("type") {
	case "One-Time", "Cumulative":
	default:
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "legacy purchase order type is invalid",
			Data: map[string]errs.CodeError{
				"type": {
					Code:    "invalid_type",
					Message: "Type must be One-Time or Cumulative for legacy purchase orders.",
				},
			},
		}
	}

	if record.IsNew() {
		record.Set("approved", time.Now())
	}

	return nil
}

func normalizeLegacyPurchaseOrderValidationError(err error) error {
	if err == nil {
		return nil
	}

	var validationErrs validation.Errors
	if errors.As(err, &validationErrs) {
		fieldErrors := make(map[string]errs.CodeError, len(validationErrs))
		for field, fieldErr := range validationErrs {
			fieldErrors[field] = errs.CodeError{
				Code:    "validation_error",
				Message: fieldErr.Error(),
			}
		}
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "validation failed",
			Data:    fieldErrors,
		}
	}

	return err
}

func ensureLegacyPurchaseOrderForEditView(record *core.Record) error {
	if record == nil {
		return &errs.HookError{
			Status:  http.StatusNotFound,
			Message: "purchase order not found",
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "po_not_found",
					Message: "purchase order not found",
				},
			},
		}
	}
	if !record.GetBool("legacy_manual_entry") {
		return &errs.HookError{
			Status:  http.StatusForbidden,
			Message: "legacy edit endpoint only supports legacy purchase orders",
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "legacy_purchase_order_only",
					Message: "legacy create/update can only be used with legacy manual purchase orders",
				},
			},
		}
	}
	return nil
}

func ensureLegacyPurchaseOrderEditable(record *core.Record) error {
	if err := ensureLegacyPurchaseOrderForEditView(record); err != nil {
		return err
	}
	switch record.GetString("status") {
	case "Closed":
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "closed legacy purchase orders cannot be edited",
			Data: map[string]errs.CodeError{
				"status": {
					Code:    "closed_legacy_purchase_order",
					Message: "closed legacy purchase orders cannot be edited",
				},
			},
		}
	case "Cancelled":
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "cancelled legacy purchase orders cannot be edited",
			Data: map[string]errs.CodeError{
				"status": {
					Code:    "cancelled_legacy_purchase_order",
					Message: "cancelled legacy purchase orders cannot be edited",
				},
			},
		}
	}
	return nil
}

func respondLegacyPurchaseOrderError(e *core.RequestEvent, err error, fallbackStatus int) error {
	var hookErr *errs.HookError
	if errors.As(err, &hookErr) {
		return e.JSON(hookErr.Status, hookErr)
	}
	if codeErr, ok := err.(*CodeError); ok {
		return e.JSON(fallbackStatus, map[string]any{
			"code":    codeErr.Code,
			"message": codeErr.Message,
		})
	}
	return e.JSON(fallbackStatus, map[string]any{
		"code":    "legacy_purchase_order_error",
		"message": err.Error(),
	})
}

func createGetLegacyPurchaseOrderForEditHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if err := requireLegacyPurchaseOrderRouteAccess(app, authRecord); err != nil {
			return respondLegacyPurchaseOrderError(e, err, http.StatusForbidden)
		}

		id := strings.TrimSpace(e.Request.PathValue("id"))
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "invalid_purchase_order_id",
				"message": "purchase order id is required",
			})
		}

		record, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]any{
				"code":    "po_not_found",
				"message": "purchase order not found",
			})
		}
		if err := ensureLegacyPurchaseOrderForEditView(record); err != nil {
			return respondLegacyPurchaseOrderError(e, err, http.StatusForbidden)
		}

		return e.JSON(http.StatusOK, record)
	}
}

func createCreateLegacyPurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth

		var payload map[string]any
		if err := e.BindBody(&payload); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "invalid_request_body",
				"message": "invalid request body",
			})
		}

		var responseRecord *core.Record
		var httpResponseStatusCode = http.StatusBadRequest
		err := app.RunInTransaction(func(txApp core.App) error {
			if err := requireLegacyPurchaseOrderRouteAccess(txApp, authRecord); err != nil {
				httpResponseStatusCode = http.StatusForbidden
				var hookErr *errs.HookError
				if errors.As(err, &hookErr) {
					httpResponseStatusCode = hookErr.Status
				}
				return err
			}

			if err := rejectLegacyPurchaseOrderDisallowedFields(payload); err != nil {
				return err
			}

			collection, err := txApp.FindCollectionByNameOrId("purchase_orders")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{Code: "purchase_orders_collection_not_found", Message: err.Error()}
			}

			record := core.NewRecord(collection)
			applyLegacyPurchaseOrderPayload(record, payload)
			if err := requireLegacyPurchaseOrderFields(record); err != nil {
				return err
			}
			if err := normalizeLegacyPurchaseOrderRecord(record); err != nil {
				return err
			}
			if err := hooks.CleanPurchaseOrder(txApp, record); err != nil {
				return err
			}
			if err := hooks.EnsureActiveDivision(txApp, record.GetString("division"), "division"); err != nil {
				return err
			}
			if err := normalizeLegacyPurchaseOrderValidationError(
				hooks.ValidatePurchaseOrder(txApp, record, true),
			); err != nil {
				return err
			}
			if err := txApp.Save(record); err != nil {
				httpResponseStatusCode = http.StatusBadRequest
				return normalizeLegacyPurchaseOrderValidationError(err)
			}

			responseRecord = record
			httpResponseStatusCode = http.StatusOK
			return nil
		})
		if err != nil {
			return respondLegacyPurchaseOrderError(e, err, httpResponseStatusCode)
		}

		return e.JSON(http.StatusOK, responseRecord)
	}
}

func createUpdateLegacyPurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		id := strings.TrimSpace(e.Request.PathValue("id"))
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "invalid_purchase_order_id",
				"message": "purchase order id is required",
			})
		}

		var payload map[string]any
		if err := e.BindBody(&payload); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "invalid_request_body",
				"message": "invalid request body",
			})
		}

		var responseRecord *core.Record
		var httpResponseStatusCode = http.StatusBadRequest
		err := app.RunInTransaction(func(txApp core.App) error {
			if err := requireLegacyPurchaseOrderRouteAccess(txApp, authRecord); err != nil {
				httpResponseStatusCode = http.StatusForbidden
				var hookErr *errs.HookError
				if errors.As(err, &hookErr) {
					httpResponseStatusCode = hookErr.Status
				}
				return err
			}

			if err := rejectLegacyPurchaseOrderDisallowedFields(payload); err != nil {
				return err
			}

			record, err := txApp.FindRecordById("purchase_orders", id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{Code: "po_not_found", Message: "purchase order not found"}
			}
			if err := ensureLegacyPurchaseOrderEditable(record); err != nil {
				httpResponseStatusCode = http.StatusBadRequest
				var hookErr *errs.HookError
				if errors.As(err, &hookErr) {
					httpResponseStatusCode = hookErr.Status
				}
				return err
			}

			applyLegacyPurchaseOrderPayload(record, payload)
			if err := requireLegacyPurchaseOrderFields(record); err != nil {
				return err
			}
			if err := normalizeLegacyPurchaseOrderRecord(record); err != nil {
				return err
			}
			if err := hooks.CleanPurchaseOrder(txApp, record); err != nil {
				return err
			}
			if err := hooks.EnsureActiveDivision(txApp, record.GetString("division"), "division"); err != nil {
				return err
			}
			if err := normalizeLegacyPurchaseOrderValidationError(
				hooks.ValidatePurchaseOrder(txApp, record, true),
			); err != nil {
				return err
			}
			if err := txApp.Save(record); err != nil {
				httpResponseStatusCode = http.StatusBadRequest
				return normalizeLegacyPurchaseOrderValidationError(err)
			}

			responseRecord = record
			httpResponseStatusCode = http.StatusOK
			return nil
		})
		if err != nil {
			return respondLegacyPurchaseOrderError(e, err, httpResponseStatusCode)
		}

		return e.JSON(http.StatusOK, responseRecord)
	}
}
