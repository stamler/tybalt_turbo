package routes

import (
	"database/sql"
	_ "embed" // Needed for //go:embed
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tybalt/constants"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed pending_pos.sql
var pendingPOsQuery string

type poApproversRequest struct {
	Division  string  `json:"division"`
	Amount    float64 `json:"amount"`
	Kind      string  `json:"kind"`
	HasJob    bool    `json:"has_job"`
	Type      string  `json:"type"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	Frequency string  `json:"frequency"`
}

type secondApproversMeta struct {
	SecondApprovalRequired  bool    `json:"second_approval_required"`
	RequesterQualifies      bool    `json:"requester_qualifies"`
	Status                  string  `json:"status"`
	ReasonCode              string  `json:"reason_code"`
	ReasonMessage           string  `json:"reason_message"`
	EvaluatedAmount         float64 `json:"evaluated_amount"`
	SecondApprovalThreshold float64 `json:"second_approval_threshold"`
	TierCeiling             float64 `json:"tier_ceiling"`
	LimitColumn             string  `json:"limit_column"`
}

type secondApproversResponse struct {
	Approvers []utilities.Approver `json:"approvers"`
	Meta      secondApproversMeta  `json:"meta"`
}

func buildSecondApproversMeta(
	app core.App,
	req poApproversRequest,
	requesterQualifies bool,
	approvers []utilities.Approver,
) (secondApproversMeta, error) {
	thresholds, err := utilities.GetPOApprovalThresholds(app)
	if err != nil {
		return secondApproversMeta{}, fmt.Errorf("error fetching approval thresholds: %w", err)
	}
	if len(thresholds) == 0 {
		return secondApproversMeta{}, fmt.Errorf("approval thresholds are not configured")
	}

	secondApprovalThreshold := thresholds[0]
	secondApprovalRequired := req.Amount > secondApprovalThreshold
	tierCeiling := constants.MAX_APPROVAL_TOTAL
	for _, threshold := range thresholds {
		if threshold >= req.Amount {
			tierCeiling = threshold
			break
		}
	}

	limitColumn := ""
	if secondApprovalRequired {
		resolvedLimitColumn, resolveErr := utilities.ResolvePOApproverLimitColumn(req.Kind, req.HasJob)
		if resolveErr != nil {
			return secondApproversMeta{}, fmt.Errorf("error resolving approver limit column: %w", resolveErr)
		}
		limitColumn = resolvedLimitColumn
	}

	meta := secondApproversMeta{
		SecondApprovalRequired:  secondApprovalRequired,
		RequesterQualifies:      requesterQualifies,
		Status:                  "required_no_candidates",
		ReasonCode:              "no_eligible_second_approvers",
		ReasonMessage:           "Second approval is required, but no eligible second approver is currently available. Assign a priority second approver.",
		EvaluatedAmount:         req.Amount,
		SecondApprovalThreshold: secondApprovalThreshold,
		TierCeiling:             tierCeiling,
		LimitColumn:             limitColumn,
	}

	switch {
	case !secondApprovalRequired:
		meta.Status = "not_required"
		meta.ReasonCode = "second_approval_not_required"
		meta.ReasonMessage = "Second approval is not required for this purchase order."
	case requesterQualifies:
		meta.Status = "requester_qualifies"
		meta.ReasonCode = "requester_is_eligible_second_approver"
		meta.ReasonMessage = "Second approval is required and the requester is eligible to perform it."
	case len(approvers) > 0:
		meta.Status = "candidates_available"
		meta.ReasonCode = "eligible_second_approvers_available"
		meta.ReasonMessage = "Eligible second approvers are available for this purchase order."
	}

	return meta, nil
}

func createApprovePurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		authRecord := e.Auth
		userId := authRecord.Id

		var httpResponseStatusCode int

		// This variable is used to track whether the original unmodified purchase
		// order has been approved. It is initialized to false and set to true if
		// the purchase order has a non-zero approved date during the transaction
		// prior to any updates. We declare the variable here so that it can be
		// used after the transaction has completed outside of the transaction
		// function.
		recordIsApproved := false
		updatedRecordIsApproved := false
		recordRequiresSecondApproval := false

		err := app.RunInTransaction(func(txApp core.App) error {
			// Fetch existing purchase order
			po, err := txApp.FindRecordById("purchase_orders", id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "po_not_found",
					Message: fmt.Sprintf("error fetching purchase order: %v", err),
				}
			}

			// Check if the purchase order is unapproved
			if po.Get("status") != "Unapproved" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_not_unapproved",
					Message: "only unapproved purchase orders can be approved",
				}
			}

			// Check if the purchase order is already rejected.
			if !po.GetDateTime("rejected").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_rejected",
					Message: "rejected purchase orders cannot be approved",
				}
			}

			/*
				The caller may not be the approver but still be qualified to approve the
				purchase order if they have a po_approver claim and the
				po_approver_props record's divisions property specifies a division that
				matches the record's division or is missing, and the amount is within
				the caller's max_amount as specified in their max_amount property on the
				po_approver_props record. In both cases, callerIsApprover is set to true
				as during the update we will set approver to the caller's uid.
			*/

			// Check if the user is an approver and/or a qualified second approver.
			// Because a caller may be a second approver without being an approver on
			// the record, we check for both.
			callerIsApprover, callerIsQualifiedSecondApprover, err := isApprover(txApp, authRecord, po)
			if err != nil {
				httpResponseStatusCode = http.StatusForbidden
				return err
			}

			thresholds, err := utilities.GetPOApprovalThresholds(txApp)
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_approval_thresholds",
					Message: fmt.Sprintf("error fetching approval thresholds: %v", err),
				}
			}
			recordIsApproved = !po.GetDateTime("approved").IsZero()
			updatedRecordIsApproved = recordIsApproved
			recordRequiresSecondApproval = po.GetFloat("approval_total") > thresholds[0]
			recordIsSecondApproved := !po.GetDateTime("second_approval").IsZero()

			// If the caller is not an approver or a qualified second approver, and
			// the PO is not already approved, return a 403 Forbidden status.
			if !recordIsApproved && !callerIsApprover && !callerIsQualifiedSecondApprover {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized_approval",
					Message: "you are not authorized to approve this purchase order",
				}
			}

			// This time will be written to the record if the approval or second
			// approval status changes
			now := time.Now()

			// If the PO is already approved and requires second approval, caller must
			// be a qualified second approver
			if recordIsApproved && recordRequiresSecondApproval && !callerIsQualifiedSecondApprover {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized_approval",
					Message: "you are not authorized to perform second approval on this purchase order",
				}
			}

			// If a PO requires second approval and caller is only doing first
			// approval, ensure second-approval ownership can be established.
			if !recordIsApproved && recordRequiresSecondApproval && !callerIsQualifiedSecondApprover {
				prioritySecondApproverID := strings.TrimSpace(po.GetString("priority_second_approver"))
				if prioritySecondApproverID == "" {
					secondApprovers, _, secondApproverErr := utilities.GetPOApprovers(
						txApp,
						nil,
						po.GetString("division"),
						po.GetFloat("approval_total"),
						po.GetString("kind"),
						po.GetString("job") != "",
						true,
					)
					if secondApproverErr != nil {
						httpResponseStatusCode = http.StatusInternalServerError
						return &CodeError{
							Code:    "error_checking_second_approvers",
							Message: fmt.Sprintf("error checking second approver availability: %v", secondApproverErr),
						}
					}
					if len(secondApprovers) == 0 {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "second_approval_unassignable",
							Message: "second approval is required, but no eligible second approver is available. Set a priority second approver before first approval.",
						}
					}
				}
			}

			// If the purchase order is not approved and the caller is an approver
			// or a qualified second approver, approve the purchase order.
			if !recordIsApproved && (callerIsApprover || callerIsQualifiedSecondApprover) {
				// Approve the purchase order
				po.Set("approved", now)
				po.Set("approver", userId)
				updatedRecordIsApproved = true
			}

			// If the purchase order is approved but requires second approval and
			// the caller is a qualified second approver, second-approve the purchase
			// order.
			if updatedRecordIsApproved && recordRequiresSecondApproval && callerIsQualifiedSecondApprover && !recordIsSecondApproved {
				po.Set("second_approval", now)
				po.Set("second_approver", userId)
				recordIsSecondApproved = true
			}

			// If both approvals are complete (or second approval wasn't needed),
			// set status and PO number
			if updatedRecordIsApproved && (!recordRequiresSecondApproval || recordIsSecondApproved) {
				po.Set("status", "Active")
				poNumber, err := GeneratePONumber(txApp, po)
				if err != nil {
					httpResponseStatusCode = http.StatusInternalServerError
					return &CodeError{
						Code:    "error_generating_po_number",
						Message: fmt.Sprintf("error generating PO number: %v", err),
					}
				}
				po.Set("po_number", poNumber)
			}

			if err := txApp.Save(po); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_updating_purchase_order",
					Message: fmt.Sprintf("error updating purchase order: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			// Check if the error is a CodeError and return the appropriate JSON response
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// return the updated purchase order from the database
		updatedPO, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Expand the new purchase order
		if errs := app.ExpandRecord(updatedPO, []string{"uid.profiles_via_uid", "approver.profiles_via_uid", "division", "job"}, nil); len(errs) > 0 {
			return e.JSON(http.StatusInternalServerError, map[string]any{
				"message": fmt.Sprintf("error expanding record: %v", errs),
				"code":    "error_expanding_record",
			})
		}

		creatorProfile := updatedPO.ExpandedOne("uid").ExpandedOne("profiles_via_uid")
		approverProfile := updatedPO.ExpandedOne("approver").ExpandedOne("profiles_via_uid")

		notificationCollection, err := app.FindCollectionByNameOrId("notifications")
		if err != nil {
			return err
		}
		var notificationRecord *core.Record = nil

		if updatedRecordIsApproved && updatedPO.GetString("priority_second_approver") != "" && updatedPO.GetString("status") != "Active" {
			// The PO is now approved but not second-approved, and the
			// priority_second_approver is set. Create a message to the
			// priority_second_approver alerting them that they need to approve the PO
			// and have a 24 hour window to do so before it is available for approval
			// by all qualified approvers.
			notificationRecord = core.NewRecord(notificationCollection)

			notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
				"code": "po_priority_second_approval_required",
			})
			if err != nil {
				return err
			}
			notificationRecord.Set("recipient", updatedPO.GetString("priority_second_approver"))
			notificationRecord.Set("template", notificationTemplate.Id)
			notificationRecord.Set("status", "pending")
			notificationRecord.Set("user", userId)
			notificationRecord.Set("data", map[string]any{
				"POId":          updatedPO.Id,
				"POCreatorName": creatorProfile.GetString("given_name") + " " + creatorProfile.GetString("surname"),
			})
		}

		if (!recordIsApproved || (recordIsApproved && recordRequiresSecondApproval)) && updatedPO.GetString("status") == "Active" && updatedPO.GetString("uid") != userId {
			// The PO was just approved (or just second approved) and is active.
			// Unless the caller is the creator of the PO (and thus would already
			// know that it has been approved), send a message to the creator
			// alerting them that the PO has been approved and is available for
			// use.
			notificationRecord = core.NewRecord(notificationCollection)

			notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
				"code": "po_active",
			})
			if err != nil {
				return err
			}

			notificationRecord.Set("recipient", updatedPO.GetString("uid"))
			notificationRecord.Set("template", notificationTemplate.Id)
			notificationRecord.Set("status", "pending")
			notificationRecord.Set("user", userId)
			notificationRecord.Set("data", map[string]any{
				"POId":           updatedPO.Id,
				"PONumber":       updatedPO.GetString("po_number"),
				"POCreatorName":  creatorProfile.GetString("given_name") + " " + creatorProfile.GetString("surname"),
				"POApproverName": approverProfile.GetString("given_name") + " " + approverProfile.GetString("surname"),
			})
		}

		// If there is a notification record to save, save it
		if notificationRecord != nil {
			if err := app.Save(notificationRecord); err != nil {
				return err
			}
		}

		// return the updated purchase order as a JSON response
		return e.JSON(http.StatusOK, updatedPO)
	}
}

// createGetPendingPurchaseOrdersHandler returns purchase orders the caller can approve now.
// It covers three cases:
// 1) First approval: status=Unapproved AND approved=” AND caller is eligible first approver for division
// 2) Priority second approver: status=Unapproved AND approved!=” AND second_approval=” AND caller == priority_second_approver
// 3) General second approver after 24h: status=Unapproved AND approved!=” AND second_approval=” AND updated < @yesterday AND caller is eligible second approver for division and amount
func createGetPendingPurchaseOrdersHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth

		rows := []map[string]any{}
		if err := app.DB().NewQuery(pendingPOsQuery).Bind(dbx.Params{
			"userId":          auth.Id,
			"poApproverClaim": constants.PO_APPROVER_CLAIM_ID,
		}).All(&rows); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_pending_pos",
				"message": fmt.Sprintf("error fetching pending purchase orders: %v", err),
			})
		}

		return e.JSON(http.StatusOK, rows)
	}
}

func createRejectPurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		/*
		   We use two separate request bindings to properly distinguish between different error cases:
		   1. First bind to a raw map to check if the rejection_reason field exists at all
		      - If the field is missing (e.g., {}), return "invalid_request_body"
		   2. Then bind to our typed struct for actual validation
		      - If the field exists but is empty/too short, return "invalid_rejection_reason"

		   This two-step process is necessary because binding directly to the struct
		   would give us an empty string for both cases, making it impossible to
		   distinguish between a missing field and an empty value.
		*/
		// First bind to a map to check if the field exists
		var rawReq map[string]interface{}
		if err := e.BindBody(&rawReq); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "you must provide a rejection reason",
				"code":    "invalid_request_body",
			})
		}

		if _, exists := rawReq["rejection_reason"]; !exists {
			return e.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "you must provide a rejection reason",
				"code":    "invalid_request_body",
			})
		}

		// Now bind to our typed struct
		var req RejectionRequest
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "you must provide a rejection reason",
				"code":    "invalid_request_body",
			})
		}

		// Validate rejection reason length
		trimmedReason := strings.TrimSpace(req.RejectionReason)
		if trimmedReason == "" || len(trimmedReason) < 5 {
			return e.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "rejection reason must be at least 5 characters",
				"code":    "invalid_rejection_reason",
			})
		}

		authRecord := e.Auth
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.RunInTransaction(func(txApp core.App) error {
			// Fetch existing purchase order
			po, err := txApp.FindRecordById("purchase_orders", id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "po_not_found",
					Message: fmt.Sprintf("error fetching purchase order: %v", err),
				}
			}

			// Check if the purchase order is already rejected.
			if !po.GetDateTime("rejected").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_rejected",
					Message: "rejected purchase orders cannot be rejected again",
				}
			}

			// Check if the purchase order is unapproved
			if po.Get("status") != "Unapproved" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_not_unapproved",
					Message: "only unapproved purchase orders can be rejected",
				}
			}

			// Check if the user is an approver and/or a qualified second approver
			callerIsApprover, callerIsQualifiedSecondApprover, err := isApprover(txApp, authRecord, po)
			if err != nil {
				return err
			}

			// Because isApprover uses GetPOApprovers, the second approvers list is
			// just users below the next threshold. This means that users above the
			// next threshold cannot reject a PO with an approval_total that exceeds
			// the threshold immediately below their max_amount.

			// If the caller is not an approver or a qualified second approver,
			// return a 403 Forbidden status. NOTE: This means that even if a
			// purchase_orders record requiring second approval is already approved,
			// it can still be rejected by any approver or a qualified second
			// approver since it isn't yet Active.
			if !(callerIsApprover || callerIsQualifiedSecondApprover) {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized_rejection",
					Message: "you are not authorized to reject this purchase order",
				}
			}

			// Update the purchase order
			po.Set("rejected", time.Now())
			po.Set("rejection_reason", req.RejectionReason)
			po.Set("rejector", userId)

			if err := txApp.Save(po); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_updating_purchase_order",
					Message: fmt.Sprintf("error updating purchase order: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// return the updated purchase order from the database
		updatedPO, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Expand the new purchase order
		if errs := app.ExpandRecord(updatedPO, []string{"uid.profiles_via_uid", "approver.profiles_via_uid", "division", "job"}, nil); len(errs) > 0 {
			return e.JSON(http.StatusInternalServerError, map[string]any{
				"message": fmt.Sprintf("error expanding record: %v", errs),
				"code":    "error_expanding_record",
			})
		}

		// Send notification to the creator (uid)
		notificationCollection, err := app.FindCollectionByNameOrId("notifications")
		if err != nil {
			// Log the error but don't fail the request, as the PO was already rejected
			app.Logger().Error("notification not sent: error finding notifications collection", "error", err)
		} else {
			notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
				"code": "po_rejected",
			})
			if err != nil {
				// Log the error but don't fail the request
				app.Logger().Error("notification not sent: error finding po_rejected notification template", "error", err)
			} else {
				notificationRecord := core.NewRecord(notificationCollection)
				notificationRecord.Set("recipient", updatedPO.GetString("uid"))
				notificationRecord.Set("template", notificationTemplate.Id)
				notificationRecord.Set("status", "pending")
				notificationRecord.Set("user", userId)
				notificationRecord.Set("data", map[string]any{
					"POId": updatedPO.Id,
				})
				if err := app.Save(notificationRecord); err != nil {
					// Log the error but don't fail the request
					app.Logger().Error("notification not sent: error saving rejection notification", "error", err)
				}
			}
		}

		return e.JSON(http.StatusOK, updatedPO)
	}
}

func createCancelPurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		userId := authRecord.Id

		var httpResponseStatusCode int
		id := e.Request.PathValue("id")

		err := app.RunInTransaction(func(txApp core.App) error {
			// Fetch existing purchase order
			po, err := txApp.FindRecordById("purchase_orders", id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "po_not_found",
					Message: fmt.Sprintf("error fetching purchase order: %v", err),
				}
			}

			// Check if the purchase order is active
			if po.Get("status") != "Active" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_not_active",
					Message: "only active purchase orders can be cancelled",
				}
			}

			// Check if the user is authorized to cancel the purchase order
			hasPayablesAdminClaim, err := utilities.HasClaim(txApp, authRecord, "payables_admin")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_user_claims",
					Message: fmt.Sprintf("error fetching user claims: %v", err),
				}
			}
			if !hasPayablesAdminClaim {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized_cancellation",
					Message: "you are not authorized to cancel this purchase order",
				}
			}

			// Count the number of associated expenses records. If there are any, the
			// purchase order cannot be cancelled.
			expenses, err := txApp.FindRecordsByFilter("expenses", "purchase_order = {:poId}", "", 0, 0, dbx.Params{
				"poId": po.Id,
			})
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_expenses",
					Message: fmt.Sprintf("error fetching expenses: %v", err),
				}
			}
			if len(expenses) > 0 {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_has_expenses",
					Message: "this purchase order has associated expenses and cannot be cancelled",
				}
			}

			// Cancel the purchase order
			po.Set("status", "Cancelled")
			po.Set("cancelled", time.Now())
			po.Set("canceller", userId)

			if err := txApp.Save(po); err != nil {
				return &CodeError{
					Code:    "error_updating_purchase_order",
					Message: fmt.Sprintf("error updating purchase order: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// return the updated purchase order from the database
		cancelledPO, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, cancelledPO)
	}
}

/*
createConvertToCumulativePurchaseOrderHandler is a function that converts a
status=Active type=One-Time purchase_orders record to a type=Cumulative
purchase_orders record. It may only be called by a user with the
payables_admin claim.
*/
func createConvertToCumulativePurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth

		var httpResponseStatusCode int
		id := e.Request.PathValue("id")

		err := app.RunInTransaction(func(txApp core.App) error {
			// Fetch existing purchase order
			po, err := txApp.FindRecordById("purchase_orders", id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "po_not_found",
					Message: fmt.Sprintf("error fetching purchase order: %v", err),
				}
			}

			// Check if the purchase order is active
			if po.Get("status") != "Active" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_not_active",
					Message: "only active purchase orders can be converted to Cumulative",
				}
			}

			// check if the purchase order has type=One-Time
			if po.GetString("type") != "One-Time" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_not_one_time",
					Message: "only One-Time purchase orders can be converted to Cumulative",
				}
			}

			// Check if the user is authorized to cancel the purchase order
			hasPayablesAdminClaim, err := utilities.HasClaim(txApp, authRecord, "payables_admin")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_user_claims",
					Message: fmt.Sprintf("error fetching user claims: %v", err),
				}
			}
			if !hasPayablesAdminClaim {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized_conversion",
					Message: "you are not authorized to convert this purchase order to Cumulative",
				}
			}

			// Update the type to Cumulative
			po.Set("type", "Cumulative")

			// Save the updated purchase order record
			if err := txApp.Save(po); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_updating_purchase_order",
					Message: fmt.Sprintf("error updating purchase order: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// return the updated purchase order from the database
		updatedPO, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, updatedPO)
	}
}

// GeneratePONumber generates a unique PO number in one of two formats:
// 1. Parent PO format: YYMM-NNNN (e.g., 2401-0001)
// 2. Child PO format:  YYMM-NNNN-XX (e.g., 2401-0001-01)
// where YY is the last two digits of the current year, MM is the current month,
// NNNN is a sequential number, and XX is a sequential suffix for child POs
// (01-99).
func GeneratePONumber(txApp core.App, record *core.Record, testDateComponents ...int) (string, error) {
	currentYear := time.Now().Year()
	currentMonth := int(time.Now().Month())
	var prefix string
	if len(testDateComponents) > 1 {
		currentYear = testDateComponents[0]
		currentMonth = testDateComponents[1]
		prefix = fmt.Sprintf("%d%02d-", currentYear%100, currentMonth)
	} else {
		prefix = fmt.Sprintf("%d%02d-", currentYear%100, currentMonth)
	}
	txApp.Logger().Debug("Generating PO number", "prefix", prefix)

	// If this is a child PO, handle differently
	if record.GetString("parent_po") != "" {
		txApp.Logger().Debug("Generating child PO number", "parent_po", record.GetString("parent_po"))
		parentId := record.GetString("parent_po")
		// don't bother storing the error since the parent will be nil both if it
		// doesn't exist and for any other error
		parent, _ := txApp.FindRecordById("purchase_orders", parentId)
		if parent == nil {
			return "", fmt.Errorf("parent PO not found")
		}
		parentNumber := parent.GetString("po_number")
		if parentNumber == "" {
			return "", fmt.Errorf("parent PO does not have a PO number")
		}

		// Find highest child suffix for this parent. Do this by filtering on
		// parentId and sorting by po_number descending then taking the first
		// record.

		// NOTE: This result will include the child PO itself (which should not yet
		// have a PO number), so we need to filter out any purchase_orders records
		// that have a po_number that is an empty string as well.
		childrenWithPONumbers, err := txApp.FindRecordsByFilter(
			"purchase_orders",
			"parent_po = {:parentId} && po_number != ''",
			"-po_number",
			1,
			0,
			dbx.Params{"parentId": parentId},
		)
		if err != nil {
			return "", fmt.Errorf("error querying child PO numbers: %v", err)
		}

		// If there are no children with a PO number, the next suffix is 1. If
		// there are children, find the highest suffix and increment it.

		nextSuffix := 1
		if len(childrenWithPONumbers) > 0 {
			lastChild := childrenWithPONumbers[0].GetString("po_number")
			suffix := lastChild[len(lastChild)-2:]

			// Convert the suffix to an integer and increment it
			fmt.Sscanf(suffix, "%d", &nextSuffix)
			nextSuffix++
		}

		if nextSuffix > 99 {
			return "", fmt.Errorf("maximum number of child POs reached (99) for parent %s", parentNumber)
		}

		childNumber := fmt.Sprintf("%s-%02d", parentNumber, nextSuffix)

		// Double check uniqueness
		existing, err := txApp.FindFirstRecordByFilter(
			"purchase_orders",
			"po_number = {:poNumber}",
			dbx.Params{"poNumber": childNumber},
		)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("error checking child PO number uniqueness: %v", err)
		}
		if existing != nil {
			return "", fmt.Errorf("generated child PO number %s already exists", childNumber)
		}

		return childNumber, nil
	}
	txApp.Logger().Debug("Generating parent PO number", "prefix", prefix)

	// Handle parent PO number generation
	// We can just filter over all POs and get the last one regardless of whether
	// it's a parent or child PO because all children POs have a parent PO with
	// a PO number and the same prefix. We do however need to filter by the current
	// year otherwise we may create a lastNumber that is sequential for a previous
	// year rather than the current year.
	existingPOs, err := txApp.FindRecordsByFilter(
		"purchase_orders",
		`po_number ~ {:prefix}`,
		"-po_number",
		1,
		0,
		dbx.Params{"prefix": prefix + "%"},
	)
	if err != nil {
		return "", fmt.Errorf("error querying existing PO numbers: %v", err)
	}

	var lastNumber int
	if len(existingPOs) > 0 {
		lastPO := existingPOs[0].GetString("po_number")
		// Extract the numeric suffix after the prefix
		numericSuffix := strings.TrimPrefix(lastPO, prefix)
		parsedNum, err := strconv.Atoi(numericSuffix)
		if err != nil {
			return "", fmt.Errorf("error parsing last PO number: %v", err)
		}
		lastNumber = parsedNum
	}
	txApp.Logger().Debug("Last PO number", "lastNumber", lastNumber)
	// Generate the new PO number
	for i := lastNumber + 1; i < 6000; i++ {
		newPONumber := fmt.Sprintf("%s%04d", prefix, i)

		// Check if the generated PO number is unique
		existing, err := txApp.FindFirstRecordByFilter(
			"purchase_orders",
			"po_number = {:poNumber}",
			dbx.Params{"poNumber": newPONumber},
		)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("error checking PO number uniqueness: %v", err)
		}

		if existing == nil {
			return newPONumber, nil
		}
	}

	return "", fmt.Errorf("unable to generate a unique PO number")
}

/*
	the isApprover function with 3 arguments: the txApp, the userId of the
	caller, and the purchase_orders record. The function performs the following
	checks and returns 2 boolean values indicating:
		1. whether the caller is permitted to approve the purchase order
		2. whether the caller is permitted to second-approve the purchase order
	We will incorporate this function into the approval logic above within the
	createApprovePurchaseOrderHandler function.
*/

func isApprover(txApp core.App, auth *core.Record, po *core.Record) (bool, bool, error) {
	kindID := po.GetString("kind")
	hasJob := po.GetString("job") != ""

	approvers, _, err := utilities.GetPOApprovers(
		txApp,
		nil,
		po.GetString("division"),
		po.GetFloat("approval_total"),
		kindID,
		hasJob,
		false,
	)
	if err != nil {
		return false, false, &CodeError{
			Code:    "error_fetching_approvers",
			Message: fmt.Sprintf("error fetching approvers: %v", err),
		}
	}

	secondApprovers, _, err := utilities.GetPOApprovers(
		txApp,
		nil,
		po.GetString("division"),
		po.GetFloat("approval_total"),
		kindID,
		hasJob,
		true,
	)
	if err != nil {
		return false, false, &CodeError{
			Code:    "error_fetching_approvers",
			Message: fmt.Sprintf("error fetching approvers: %v", err),
		}
	}

	callerIsApprover := false
	callerIsQualifiedSecondApprover := false

	for _, approver := range approvers {
		if approver.ID == auth.Id {
			callerIsApprover = true
			break
		}
	}

	for _, secondApprover := range secondApprovers {
		if secondApprover.ID == auth.Id {
			callerIsQualifiedSecondApprover = true
			break
		}
	}

	return callerIsApprover, callerIsQualifiedSecondApprover, nil
}

func parseApproversRequest(e *core.RequestEvent) (poApproversRequest, error) {
	req := poApproversRequest{
		Kind: utilities.DefaultExpenditureKindID(),
	}
	q := e.Request.URL.Query()
	req.Division = q.Get("division")
	amountStr := q.Get("amount")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return req, fmt.Errorf("invalid amount")
	}
	req.Amount = amount
	req.Type = q.Get("type")
	req.StartDate = q.Get("start_date")
	req.EndDate = q.Get("end_date")
	req.Frequency = q.Get("frequency")
	if kind := strings.TrimSpace(q.Get("kind")); kind != "" {
		req.Kind = kind
	}
	if hasJobRaw := strings.TrimSpace(q.Get("has_job")); hasJobRaw != "" {
		parsedHasJob, parseErr := strconv.ParseBool(hasJobRaw)
		if parseErr != nil {
			return req, fmt.Errorf("invalid has_job")
		}
		req.HasJob = parsedHasJob
	}

	return req, nil
}

// createGetApproversHandler returns first- or second-approver candidates.
func createGetApproversHandler(app core.App, forSecondApproval bool) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth

		req, err := parseApproversRequest(e)
		if err != nil {
			switch err.Error() {
			case "invalid amount":
				return e.JSON(http.StatusBadRequest, map[string]string{
					"code":    "invalid_amount",
					"message": "Amount must be a valid number",
				})
			case "invalid has_job":
				return e.JSON(http.StatusBadRequest, map[string]string{
					"code":    "invalid_has_job",
					"message": "has_job must be a valid boolean",
				})
			default:
				return e.JSON(http.StatusBadRequest, map[string]string{
					"code":    "invalid_request_query",
					"message": "request query must be valid",
				})
			}
		}

		if strings.TrimSpace(req.Division) == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"code":    "invalid_division",
				"message": "division is required",
			})
		}
		if strings.TrimSpace(req.Kind) == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"code":    "invalid_kind",
				"message": "kind is required",
			})
		}
		req.Kind = utilities.NormalizeExpenditureKindID(req.Kind)

		// Check for recurring purchase order query parameters and calculate the total value if necessary
		if req.Type == "Recurring" {
			req.Amount, err = calculateRecurringPurchaseOrderTotalValue(
				app,
				req.Amount,
				req.StartDate,
				req.EndDate,
				req.Frequency,
			)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"code":    "invalid_parameters",
					"message": fmt.Sprintf("Error calculating recurring PO total: %v", err),
				})
			}
		}

		approvers, requesterQualifies, err := utilities.GetPOApprovers(
			app,
			auth,
			req.Division,
			req.Amount,
			req.Kind,
			req.HasJob,
			forSecondApproval,
		)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_approvers",
				"message": fmt.Sprintf("Error fetching approvers: %v", err),
			})
		}

		if forSecondApproval {
			meta, metaErr := buildSecondApproversMeta(app, req, requesterQualifies, approvers)
			if metaErr != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"code":    "error_building_second_approver_meta",
					"message": fmt.Sprintf("Error building second approver metadata: %v", metaErr),
				})
			}

			return e.JSON(http.StatusOK, secondApproversResponse{
				Approvers: approvers,
				Meta:      meta,
			})
		}

		return e.JSON(http.StatusOK, approvers)
	}
}

// calculate the total value of a recurring purchase order this is used to
// determine the approvers for the purchase order it is used in the getApprovers
// and getSecondApprovers handlers it is also used in the createPurchaseOrder
// handler to validate the total value of the purchase order. It is a wrapper
// around CalculateRecurringPurchaseOrderTotalValue function that assembles
// query parameters into a temporary purchase_orders record.
func calculateRecurringPurchaseOrderTotalValue(app core.App, amount float64, startDate string, endDate string, frequency string) (float64, error) {
	// Validate required parameters
	if startDate == "" || endDate == "" || frequency == "" {
		return 0, fmt.Errorf("start_date, end_date, and frequency are required for recurring purchase orders")
	}

	// Create a temporary record for calculation
	tempPO := core.NewRecord(core.NewCollection("purchase_orders", "purchase_orders"))
	tempPO.Set("date", startDate)
	tempPO.Set("end_date", endDate)
	tempPO.Set("frequency", frequency)
	tempPO.Set("total", amount)

	// Calculate the actual total for recurring PO
	_, calculatedTotal, err := utilities.CalculateRecurringPurchaseOrderTotalValue(app, tempPO)
	if err != nil {
		return 0, fmt.Errorf("Error calculating recurring PO total: %v", err)
	}

	return calculatedTotal, nil
}
