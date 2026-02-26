package routes

import (
	"database/sql"
	_ "embed" // Needed for //go:embed
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tybalt/constants"
	"tybalt/notifications"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const poVisibilityBaseToken = "__PO_VISIBILITY_BASE__"

//go:embed po_visibility_base.sql
var poVisibilityBaseQuery string

//go:embed pending_pos.sql
var pendingPOsQueryTemplate string

//go:embed pending_po_by_id.sql
var pendingPOByIDQueryTemplate string

//go:embed visible_pos.sql
var visiblePOsQueryTemplate string

//go:embed visible_po_by_id.sql
var visiblePOByIDQueryTemplate string

var (
	pendingPOsQuery    string
	pendingPOByIDQuery string
	visiblePOsQuery    string
	visiblePOByIDQuery string
)

func init() {
	pendingPOsQuery = strings.ReplaceAll(pendingPOsQueryTemplate, poVisibilityBaseToken, poVisibilityBaseQuery)
	pendingPOByIDQuery = strings.ReplaceAll(pendingPOByIDQueryTemplate, poVisibilityBaseToken, poVisibilityBaseQuery)
	visiblePOsQuery = strings.ReplaceAll(visiblePOsQueryTemplate, poVisibilityBaseToken, poVisibilityBaseQuery)
	visiblePOByIDQuery = strings.ReplaceAll(visiblePOByIDQueryTemplate, poVisibilityBaseToken, poVisibilityBaseQuery)
}

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
	LimitColumn             string  `json:"limit_column"`
	SecondStageTimeoutHours float64 `json:"second_stage_timeout_hours"`
}

type secondApproversResponse struct {
	Approvers []utilities.Approver `json:"approvers"`
	Meta      secondApproversMeta  `json:"meta"`
}

type purchaseOrderVisibilityRow struct {
	ID                         string  `db:"id" json:"id"`
	PONumber                   string  `db:"po_number" json:"po_number"`
	Status                     string  `db:"status" json:"status"`
	UID                        string  `db:"uid" json:"uid"`
	Type                       string  `db:"type" json:"type"`
	Date                       string  `db:"date" json:"date"`
	EndDate                    string  `db:"end_date" json:"end_date"`
	Frequency                  string  `db:"frequency" json:"frequency"`
	Division                   string  `db:"division" json:"division"`
	Description                string  `db:"description" json:"description"`
	Total                      float64 `db:"total" json:"total"`
	PaymentType                string  `db:"payment_type" json:"payment_type"`
	Attachment                 string  `db:"attachment" json:"attachment"`
	Rejector                   string  `db:"rejector" json:"rejector"`
	Rejected                   string  `db:"rejected" json:"rejected"`
	RejectionReason            string  `db:"rejection_reason" json:"rejection_reason"`
	Approver                   string  `db:"approver" json:"approver"`
	Approved                   string  `db:"approved" json:"approved"`
	SecondApprover             string  `db:"second_approver" json:"second_approver"`
	SecondApproval             string  `db:"second_approval" json:"second_approval"`
	Canceller                  string  `db:"canceller" json:"canceller"`
	Cancelled                  string  `db:"cancelled" json:"cancelled"`
	Job                        string  `db:"job" json:"job"`
	Category                   string  `db:"category" json:"category"`
	Kind                       string  `db:"kind" json:"kind"`
	Vendor                     string  `db:"vendor" json:"vendor"`
	ParentPO                   string  `db:"parent_po" json:"parent_po"`
	Created                    string  `db:"created" json:"created"`
	Updated                    string  `db:"updated" json:"updated"`
	Closer                     string  `db:"closer" json:"closer"`
	Closed                     string  `db:"closed" json:"closed"`
	ClosedBySystem             bool    `db:"closed_by_system" json:"closed_by_system"`
	PrioritySecondApprover     string  `db:"priority_second_approver" json:"priority_second_approver"`
	ApprovalTotal              float64 `db:"approval_total" json:"approval_total"`
	CommittedExpensesCount     int     `db:"committed_expenses_count" json:"committed_expenses_count"`
	ExpensesTotal              float64 `db:"expenses_total" json:"expenses_total"`
	RecurringExpectedCount     int     `db:"recurring_expected_occurrences" json:"recurring_expected_occurrences"`
	RecurringRemainingCount    int     `db:"recurring_remaining_occurrences" json:"recurring_remaining_occurrences"`
	CumulativeRemainingBalance float64 `db:"cumulative_remaining_balance" json:"cumulative_remaining_balance"`
	UIDName                    string  `db:"uid_name" json:"uid_name"`
	ApproverName               string  `db:"approver_name" json:"approver_name"`
	SecondApproverName         string  `db:"second_approver_name" json:"second_approver_name"`
	PrioritySecondApproverName string  `db:"priority_second_approver_name" json:"priority_second_approver_name"`
	RejectorName               string  `db:"rejector_name" json:"rejector_name"`
	ParentPONumber             string  `db:"parent_po_number" json:"parent_po_number"`
	VendorName                 string  `db:"vendor_name" json:"vendor_name"`
	VendorAlias                string  `db:"vendor_alias" json:"vendor_alias"`
	JobNumber                  string  `db:"job_number" json:"job_number"`
	ClientName                 string  `db:"client_name" json:"client_name"`
	ClientID                   string  `db:"client_id" json:"client_id"`
	JobDescription             string  `db:"job_description" json:"job_description"`
	DivisionCode               string  `db:"division_code" json:"division_code"`
	DivisionName               string  `db:"division_name" json:"division_name"`
	CategoryName               string  `db:"category_name" json:"category_name"`
}

func buildSecondApproversMeta(
	app core.App,
	requesterQualifies bool,
	approvers []utilities.Approver,
	policy utilities.POApproverPolicy,
	amount float64,
) secondApproversMeta {
	meta := secondApproversMeta{
		SecondApprovalRequired:  policy.SecondApprovalRequired,
		RequesterQualifies:      requesterQualifies,
		Status:                  "required_no_candidates",
		ReasonCode:              "no_eligible_second_approvers",
		ReasonMessage:           "Second approval is required, but no second-stage approver can final-approve this amount.",
		EvaluatedAmount:         amount,
		SecondApprovalThreshold: policy.SecondApprovalThreshold,
		LimitColumn:             policy.LimitColumn,
		SecondStageTimeoutHours: utilities.GetPurchaseOrderSecondStageTimeoutHours(app),
	}

	switch {
	case !policy.SecondApprovalRequired:
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
		meta.ReasonMessage = "Eligible second approvers who can final-approve this amount are available for this purchase order."
	}

	return meta
}

func purchaseOrderVisibilityParams(
	app core.App,
	userID string,
	scope string,
	staleBefore string,
	expiringBefore string,
) dbx.Params {
	return dbx.Params{
		"userId":          userID,
		"poApproverClaim": constants.PO_APPROVER_CLAIM_ID,
		"timeoutHours":    utilities.GetPurchaseOrderSecondStageTimeoutHours(app),
		"scope":           scope,
		"staleBefore":     staleBefore,
		"expiringBefore":  expiringBefore,
	}
}

func createApprovePurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		authRecord := e.Auth
		userId := authRecord.Id

		var httpResponseStatusCode int
		recordWasFirstApproved := false
		recordNowFirstApproved := false
		recordRequiresSecondApproval := false
		recordActivated := false

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

			hasJob := strings.TrimSpace(po.GetString("job")) != ""
			kindID := utilities.NormalizeExpenditureKindID(po.GetString("kind"), hasJob)
			approvalTotal := po.GetFloat("approval_total")

			policy, err := utilities.GetPOApproverPolicy(
				txApp,
				po.GetString("division"),
				approvalTotal,
				kindID,
				hasJob,
			)
			if err != nil {
				if errors.Is(err, utilities.ErrUnknownExpenditureKind) {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "invalid_expenditure_kind",
						Message: "purchase order kind is invalid or no longer exists",
					}
				}
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_computing_approval_policy",
					Message: fmt.Sprintf("error computing approval policy: %v", err),
				}
			}

			recordRequiresSecondApproval = policy.SecondApprovalRequired
			recordWasFirstApproved = !po.GetDateTime("approved").IsZero()
			recordNowFirstApproved = recordWasFirstApproved
			recordIsSecondApproved := !po.GetDateTime("second_approval").IsZero()

			now := time.Now()
			assignedApproverID := strings.TrimSpace(po.GetString("approver"))

			activateRecord := func() error {
				po.Set("status", "Active")
				if strings.TrimSpace(po.GetString("po_number")) != "" {
					return nil
				}
				poNumber, poNumberErr := GeneratePONumber(txApp, po)
				if poNumberErr != nil {
					httpResponseStatusCode = http.StatusInternalServerError
					return &CodeError{
						Code:    "error_generating_po_number",
						Message: fmt.Sprintf("error generating PO number: %v", poNumberErr),
					}
				}
				po.Set("po_number", poNumber)
				return nil
			}

			// Stage 1 or bypass path.
			if !recordWasFirstApproved {
				// Combined dual approval (bypass fast path).
				if recordRequiresSecondApproval && policy.IsSecondStageApprover(userId) {
					if !policy.HasSufficientFinalLimit(userId, approvalTotal) {
						httpResponseStatusCode = http.StatusForbidden
						return &CodeError{
							Code:    "insufficient_final_limit",
							Message: "you do not have sufficient approval limit to complete final approval",
						}
					}
					po.Set("approver", userId)
					po.Set("approved", now)
					po.Set("second_approver", userId)
					po.Set("second_approval", now)
					recordNowFirstApproved = true
					recordActivated = true
					if err := activateRecord(); err != nil {
						return err
					}
				} else {
					if userId != assignedApproverID {
						httpResponseStatusCode = http.StatusForbidden
						return &CodeError{
							Code:    "unauthorized_approval",
							Message: "you are not authorized to approve this purchase order",
						}
					}

					if recordRequiresSecondApproval && len(policy.SecondStageApprovers) == 0 {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "second_pool_empty",
							Message: "second approval pool is empty for this purchase order; contact an administrator",
						}
					}

					if recordRequiresSecondApproval {
						prioritySecondApproverID := strings.TrimSpace(po.GetString("priority_second_approver"))
						if prioritySecondApproverID == "" {
							httpResponseStatusCode = http.StatusBadRequest
							return &CodeError{
								Code:    "priority_second_approver_required",
								Message: "priority second approver is required when second approval is required",
							}
						}
						if !policy.IsSecondStageApprover(prioritySecondApproverID) {
							httpResponseStatusCode = http.StatusBadRequest
							return &CodeError{
								Code:    "invalid_priority_second_approver_for_stage",
								Message: "priority second approver is not valid for second-stage approval",
							}
						}
					}

					if recordRequiresSecondApproval && len(policy.FirstStageApprovers) == 0 {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "first_pool_empty",
							Message: "first approval pool is empty for this purchase order; contact an administrator",
						}
					}

					if assignedApproverID == "" || !policy.IsFirstStageApprover(assignedApproverID) {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "invalid_approver_for_stage",
							Message: "assigned approver is not valid for first-stage approval",
						}
					}

					po.Set("approved", now)
					po.Set("approver", userId)
					recordNowFirstApproved = true

					if !recordRequiresSecondApproval {
						recordActivated = true
						if err := activateRecord(); err != nil {
							return err
						}
					}
				}
			} else {
				// Stage 2 (final) path for dual-required records.
				if recordRequiresSecondApproval {
					if recordIsSecondApproved {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "po_not_unapproved",
							Message: "only unapproved purchase orders can be approved",
						}
					}
					if !policy.IsSecondStageApprover(userId) {
						httpResponseStatusCode = http.StatusForbidden
						return &CodeError{
							Code:    "unauthorized_approval",
							Message: "you are not authorized to perform second approval on this purchase order",
						}
					}
					if !policy.HasSufficientFinalLimit(userId, approvalTotal) {
						httpResponseStatusCode = http.StatusForbidden
						return &CodeError{
							Code:    "insufficient_final_limit",
							Message: "you do not have sufficient approval limit to complete final approval",
						}
					}

					po.Set("second_approval", now)
					po.Set("second_approver", userId)
					recordActivated = true
					if err := activateRecord(); err != nil {
						return err
					}
				} else {
					// Single-stage PO that was first-approved but never activated.
					if assignedApproverID == "" || !policy.IsFirstStageApprover(assignedApproverID) {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "invalid_approver_for_stage",
							Message: "assigned approver is not valid for first-stage approval",
						}
					}
					if userId != assignedApproverID && !policy.IsFirstStageApprover(userId) {
						httpResponseStatusCode = http.StatusForbidden
						return &CodeError{
							Code:    "unauthorized_approval",
							Message: "you are not authorized to approve this purchase order",
						}
					}
					recordActivated = true
					if err := activateRecord(); err != nil {
						return err
					}
				}
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

		if !recordWasFirstApproved && recordNowFirstApproved && recordRequiresSecondApproval && updatedPO.GetString("priority_second_approver") != "" && updatedPO.GetString("status") != "Active" {
			// The PO is now approved but not second-approved, and the
			// priority_second_approver is set. Create a message to the
			// priority_second_approver alerting them that they need to approve the PO
			// and have an exclusive window to do so before it is available for approval
			// by all qualified approvers.
			_, err := notifications.DispatchNotification(app, notifications.DispatchArgs{
				TemplateCode: "po_priority_second_approval_required",
				RecipientUID: updatedPO.GetString("priority_second_approver"),
				Data: map[string]any{
					"POId":          updatedPO.Id,
					"POCreatorName": creatorProfile.GetString("given_name") + " " + creatorProfile.GetString("surname"),
					"ActionURL":     notifications.BuildActionURL(app, fmt.Sprintf("/pos/%s/edit", updatedPO.Id)),
				},
				System:   false,
				ActorUID: userId,
				Mode:     notifications.DeliveryImmediate,
			})
			if err != nil {
				return err
			}
		} else if recordActivated && updatedPO.GetString("status") == "Active" && updatedPO.GetString("uid") != userId {
			// The PO was just approved (or just second approved) and is active.
			// Unless the caller is the creator of the PO (and thus would already
			// know that it has been approved), send a message to the creator
			// alerting them that the PO has been approved and is available for
			// use.
			_, err := notifications.DispatchNotification(app, notifications.DispatchArgs{
				TemplateCode: "po_active",
				RecipientUID: updatedPO.GetString("uid"),
				Data: map[string]any{
					"POId":           updatedPO.Id,
					"PONumber":       updatedPO.GetString("po_number"),
					"POCreatorName":  creatorProfile.GetString("given_name") + " " + creatorProfile.GetString("surname"),
					"POApproverName": approverProfile.GetString("given_name") + " " + approverProfile.GetString("surname"),
					"ActionURL":      notifications.BuildActionURL(app, fmt.Sprintf("/pos/%s/details", updatedPO.Id)),
				},
				System:   false,
				ActorUID: userId,
				Mode:     notifications.DeliveryImmediate,
			})
			if err != nil {
				return err
			}
		}

		// return the updated purchase order as a JSON response
		return e.JSON(http.StatusOK, updatedPO)
	}
}

// createGetPendingPurchaseOrdersHandler returns purchase orders the caller can approve now.
// It covers:
// 1) Stage 1: status=Unapproved, first approval pending, caller is assigned first approver.
// 2) Stage 2 exclusive window: first approved, second pending, caller is priority second approver.
// 3) Stage 2 general visibility after timeout T: first approved, second pending, caller is an eligible second approver.
func createGetPendingPurchaseOrdersHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth

		rows := []purchaseOrderVisibilityRow{}
		if err := app.DB().NewQuery(pendingPOsQuery).Bind(
			purchaseOrderVisibilityParams(app, auth.Id, "all", "", ""),
		).All(&rows); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_pending_pos",
				"message": fmt.Sprintf("error fetching pending purchase orders: %v", err),
			})
		}

		return e.JSON(http.StatusOK, rows)
	}
}

func createGetPendingPurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		id := strings.TrimSpace(e.Request.PathValue("id"))
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"code":    "invalid_purchase_order_id",
				"message": "purchase order id is required",
			})
		}

		rows := []purchaseOrderVisibilityRow{}
		params := purchaseOrderVisibilityParams(app, auth.Id, "all", "", "")
		params["id"] = id

		if err := app.DB().NewQuery(pendingPOByIDQuery).Bind(params).All(&rows); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_pending_po",
				"message": fmt.Sprintf("error fetching pending purchase order: %v", err),
			})
		}
		if len(rows) == 0 {
			return e.JSON(http.StatusNotFound, map[string]string{
				"code":    "pending_po_not_found",
				"message": "pending purchase order not found",
			})
		}

		return e.JSON(http.StatusOK, rows[0])
	}
}

func createGetVisiblePurchaseOrdersHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth

		scope := strings.ToLower(strings.TrimSpace(e.Request.URL.Query().Get("scope")))
		if scope == "" {
			scope = "all"
		}

		switch scope {
		case "all", "mine", "active", "rejected", "stale", "expiring":
		default:
			return e.JSON(http.StatusBadRequest, map[string]string{
				"code":    "invalid_scope",
				"message": "scope must be one of: all, mine, active, rejected, stale, expiring",
			})
		}

		staleBefore := strings.TrimSpace(e.Request.URL.Query().Get("stale_before"))
		if scope == "stale" && staleBefore == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"code":    "missing_stale_before",
				"message": "stale_before is required when scope=stale",
			})
		}
		expiringBefore := strings.TrimSpace(e.Request.URL.Query().Get("expiring_before"))
		if scope == "expiring" && expiringBefore == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"code":    "missing_expiring_before",
				"message": "expiring_before is required when scope=expiring",
			})
		}

		rows := []purchaseOrderVisibilityRow{}
		if err := app.DB().NewQuery(visiblePOsQuery).Bind(
			purchaseOrderVisibilityParams(app, auth.Id, scope, staleBefore, expiringBefore),
		).All(&rows); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_visible_pos",
				"message": fmt.Sprintf("error fetching visible purchase orders: %v", err),
			})
		}

		return e.JSON(http.StatusOK, rows)
	}
}

func createGetVisiblePurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		id := strings.TrimSpace(e.Request.PathValue("id"))
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"code":    "invalid_purchase_order_id",
				"message": "purchase order id is required",
			})
		}

		rows := []purchaseOrderVisibilityRow{}
		params := purchaseOrderVisibilityParams(app, auth.Id, "all", "", "")
		params["id"] = id

		if err := app.DB().NewQuery(visiblePOByIDQuery).Bind(params).All(&rows); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_visible_po",
				"message": fmt.Sprintf("error fetching visible purchase order: %v", err),
			})
		}
		if len(rows) == 0 {
			return e.JSON(http.StatusNotFound, map[string]string{
				"code":    "po_not_found_or_not_visible",
				"message": "purchase order not found or not visible",
			})
		}

		return e.JSON(http.StatusOK, rows[0])
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

			hasJob := po.GetString("job") != ""
			kindID := utilities.NormalizeExpenditureKindID(po.GetString("kind"), hasJob)
			policy, err := utilities.GetPOApproverPolicy(
				txApp,
				po.GetString("division"),
				po.GetFloat("approval_total"),
				kindID,
				hasJob,
			)
			if err != nil {
				if errors.Is(err, utilities.ErrUnknownExpenditureKind) {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "invalid_expenditure_kind",
						Message: "purchase order kind is invalid or no longer exists",
					}
				}
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_computing_approval_policy",
					Message: fmt.Sprintf("error computing purchase order approval policy: %v", err),
				}
			}

			callerIsApprover := policy.IsFirstStageApprover(authRecord.Id)
			callerIsQualifiedSecondApprover := policy.IsSecondStageApprover(authRecord.Id)

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
		if _, err := notifications.DispatchNotification(app, notifications.DispatchArgs{
			TemplateCode: "po_rejected",
			RecipientUID: updatedPO.GetString("uid"),
			Data: map[string]any{
				"POId":            updatedPO.Id,
				"POUrl":           fmt.Sprintf("/pos/%s/details", updatedPO.Id),
				"PONumber":        updatedPO.GetString("po_number"),
				"RejectionReason": updatedPO.GetString("rejection_reason"),
				"ActionURL":       notifications.BuildActionURL(app, fmt.Sprintf("/pos/%s/details", updatedPO.Id)),
			},
			System:   false,
			ActorUID: userId,
			Mode:     notifications.DeliveryImmediate,
		}); err != nil {
			// Log the error but don't fail the request, as the PO was already rejected
			app.Logger().Error("notification not sent: error creating rejection notification", "error", err)
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
	if len(testDateComponents) > 1 {
		currentYear = testDateComponents[0]
		currentMonth = testDateComponents[1]
	}
	prefix := fmt.Sprintf("%d%02d-", currentYear%100, currentMonth)

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
	// Filter for parent POs only (parent_po = '') with the current month prefix,
	// excluding manually assigned/imported numbers (5000+).
	//
	// NOTE: Manually assigned/imported PO numbers start at YYMM-5000+. We reserve
	// 5000+ and only auto-generate numbers below that threshold.
	upperBound := fmt.Sprintf("%s%04d", prefix, 5000)
	existingPOs, err := txApp.FindRecordsByFilter(
		"purchase_orders",
		`parent_po = '' && po_number ~ {:like} && po_number < {:upperBound}`,
		"-po_number",
		1,
		0,
		dbx.Params{
			"like":       prefix + "%",
			"upperBound": upperBound,
		},
	)
	if err != nil {
		return "", fmt.Errorf("error querying existing PO numbers: %v", err)
	}

	var lastNumber int
	if len(existingPOs) > 0 {
		lastPO := existingPOs[0].GetString("po_number")
		// Extract the numeric suffix after the prefix.
		//
		// Even though we filter out child POs (`parent_po = ''`), we defensively
		// strip any trailing "-XX" segment if it exists (e.g. if historical/manual
		// data contains dashed suffixes). This avoids parse failures like
		// `strconv.Atoi("0010-01")`.
		numericSuffix := strings.TrimPrefix(lastPO, prefix)
		numericSuffix, _, _ = strings.Cut(numericSuffix, "-")
		parsedNum, err := strconv.Atoi(numericSuffix)
		if err != nil {
			return "", fmt.Errorf("error parsing last PO number: %v", err)
		}
		lastNumber = parsedNum
	}
	txApp.Logger().Debug("Last PO number", "lastNumber", lastNumber)
	// Generate the new PO number
	for i := lastNumber + 1; i < 5000; i++ {
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

func parseApproversRequest(e *core.RequestEvent) (poApproversRequest, error) {
	req := poApproversRequest{
		Kind: utilities.DefaultCapitalExpenditureKindID(),
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
		req.Kind = utilities.NormalizeExpenditureKindID(req.Kind, req.HasJob)

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

		policy, err := utilities.GetPOApproverPolicy(
			app,
			req.Division,
			req.Amount,
			req.Kind,
			req.HasJob,
		)
		if err != nil {
			if errors.Is(err, utilities.ErrUnknownExpenditureKind) {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"code":    "invalid_kind",
					"message": "kind is invalid or no longer exists",
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_approvers",
				"message": fmt.Sprintf("Error fetching approvers: %v", err),
			})
		}

		if forSecondApproval {
			approvers := policy.SecondStageApprovers
			requesterQualifies := auth != nil && policy.IsSecondStageApprover(auth.Id)
			if requesterQualifies && policy.SecondApprovalRequired {
				// Preserve prior endpoint behavior for self-qualified requester:
				// empty list indicates UI auto-self handling.
				approvers = []utilities.Approver{}
			}
			meta := buildSecondApproversMeta(app, requesterQualifies, approvers, policy, req.Amount)
			if meta.SecondApprovalRequired && !requesterQualifies && len(approvers) == 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"code":    "second_pool_empty",
					"message": "no second-stage approvers can final-approve this amount; contact an administrator",
				})
			}

			return e.JSON(http.StatusOK, secondApproversResponse{
				Approvers: approvers,
				Meta:      meta,
			})
		}

		return e.JSON(http.StatusOK, policy.FirstStageApprovers)
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
