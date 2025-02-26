package routes

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// recordFinder defines the minimal interface needed for PO number generation operations.
//
// This interface exists for two main reasons:
//  1. Interface Segregation: It specifies only the database operations required for PO number
//     generation, making the code's dependencies explicit and minimal.
//  2. Testability: It enables testing of PO number generation logic without requiring a full
//     database connection. The production code uses *daos.Dao while tests can use a mock
//     implementation that only implements these specific methods.
//
// Note: While this interface is primarily useful for testing, it lives in the production
// code (not in test packages) because it represents a real business capability that the
// production code depends on. This maintains proper dependency direction - tests depend
// on production code, not vice versa.
type recordFinder interface {
	FindRecordById(collectionModelOrIdentifier any, id string, expands ...func(*dbx.SelectQuery) error) (*core.Record, error)
	FindRecordsByFilter(collectionModelOrIdentifier any, filter string, sort string, limit int, offset int, params ...dbx.Params) ([]*core.Record, error)
	FindFirstRecordByFilter(collectionModelOrIdentifier any, filter string, params ...dbx.Params) (*core.Record, error)
}

func createApprovePurchaseOrderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

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
				The caller may not be the approver but still be qualified to approve
				the purchase order if they have a po_approver claim and the payload
				specifies a division that matches the record's division. In either case,
				callerIsApprover is set to true as during the update we will set approver
				to the caller's uid.
			*/

			// Check if the user is an approver and/or a qualified second approver.
			// Because a caller may be a second approver without being an approver on
			// the record, we check for both.
			callerIsApprover, callerIsQualifiedSecondApprover, err := isApprover(txApp, authRecord, po)
			if err != nil {
				return err
			}

			// If the caller is not an approver or a qualified second approver,
			// return a 403 Forbidden status.
			if !callerIsApprover && !callerIsQualifiedSecondApprover {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized_approval",
					Message: "you are not authorized to approve this purchase order",
				}
			}

			recordIsApproved := !po.GetDateTime("approved").IsZero()
			recordRequiresSecondApproval := po.GetString("second_approver_claim") != ""
			recordIsSecondApproved := !po.GetDateTime("second_approval").IsZero()

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

			// If the purchase order is not approved and the caller is an approver
			// or a qualified second approver, approve the purchase order.
			if !recordIsApproved && (callerIsApprover || callerIsQualifiedSecondApprover) {
				// Approve the purchase order
				po.Set("approved", now)
				po.Set("approver", userId)
				recordIsApproved = true
			}

			// If the purchase order is approved but requires second approval and
			// the caller is a qualified second approver, second-approve the purchase
			// order.
			if recordIsApproved && recordRequiresSecondApproval && callerIsQualifiedSecondApprover && !recordIsSecondApproved {
				po.Set("second_approval", now)
				po.Set("second_approver", userId)
				recordIsSecondApproved = true
			}

			// If both approvals are complete (or second approval wasn't needed),
			// set status and PO number
			if recordIsApproved && (!recordRequiresSecondApproval || recordIsSecondApproved) {
				po.Set("status", "Active")
				poNumber, err := GeneratePONumber(txApp, po)
				if err != nil {
					return &CodeError{
						Code:    "error_generating_po_number",
						Message: fmt.Sprintf("error generating PO number: %v", err),
					}
				}
				po.Set("po_number", poNumber)
			}

			if err := txApp.Save(po); err != nil {
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
			return e.JSON(http.StatusInternalServerError, map[string]interface{}{
				"message": fmt.Sprintf("error expanding record: %v", errs),
				"code":    "error_expanding_record",
			})
		}
		// return the updated purchase order as a JSON response
		return e.JSON(http.StatusOK, updatedPO)
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

			// TODO: send notification (email) to the creator (uid), and approver
			// that the purchase order has been rejected, at what time, and by whom
			// This will be implemented through the notification service that hasn't
			// been built yet.
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
status=Active type=Normal purchase_orders record to a type=Cumulative
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

			// check if the purchase order has type=Normal
			if po.GetString("type") != "Normal" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_not_normal",
					Message: "only Normal purchase orders can be converted to Cumulative",
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
// 1. Parent PO format: YYYY-NNNN (e.g., 2024-0001)
// 2. Child PO format:  YYYY-NNNN-XX (e.g., 2024-0001-01)
// where YYYY is the current year, NNNN is a sequential number,
// and XX is a sequential suffix for child POs (01-99).
func GeneratePONumber(txApp recordFinder, record *core.Record, testYear ...int) (string, error) {
	currentYear := time.Now().Year()
	if len(testYear) > 0 {
		currentYear = testYear[0]
	}
	prefix := fmt.Sprintf("%d-", currentYear)

	// If this is a child PO, handle differently
	if record.GetString("parent_po") != "" {
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
		children, err := txApp.FindRecordsByFilter(
			"purchase_orders",
			"parent_po = {:parentId}",
			"-po_number",
			1,
			0,
			dbx.Params{"parentId": parentId},
		)
		if err != nil {
			return "", fmt.Errorf("error querying child PO numbers: %v", err)
		}

		// If there are no children, the next suffix is 1. If there are children,
		// find the highest suffix and increment it.

		nextSuffix := 1
		if len(children) > 0 {
			lastChild := children[0].GetString("po_number")
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

	// Handle parent PO number generation
	// We can just filter over all POs and get the last one regardless of whether
	// it's a parent or child PO because all children POs have a parent PO with
	// a PO number and the same prefix. We do however need to filter by the current
	// year otherwise we may create a lastNumber that is sequential for a previous
	// year rather than the current year.
	existingPOs, err := txApp.FindRecordsByFilter(
		"purchase_orders",
		`po_number ~ '{:current_year}-%'`,
		"-po_number",
		1,
		0,
		dbx.Params{"current_year": currentYear},
	)
	if err != nil {
		return "", fmt.Errorf("error querying existing PO numbers: %v", err)
	}

	var lastNumber int
	if len(existingPOs) > 0 {
		lastPO := existingPOs[0].Get("po_number").(string)
		_, err := fmt.Sscanf(lastPO, "%d-%04d", &currentYear, &lastNumber)
		if err != nil {
			return "", fmt.Errorf("error parsing last PO number: %v", err)
		}
	}

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
	// Check if the caller is the approver specified in the record
	callerIsApprover := po.GetString("approver") == auth.Id

	// if the caller is not the approver on the record, perform additional checks
	// to see if the caller is nontheless permitted to approve the purchase order
	if !callerIsApprover {

		// Check if the caller is a qualified approver (has the po_approver claim
		// and that claim has a payload that includes the division of the purchase
		// order)
		hasPoApproverClaim, err := utilities.HasClaim(txApp, auth, "po_approver")
		if err != nil {
			return false, false, &CodeError{
				Code:    "error_checking_po_approver_claim",
				Message: fmt.Sprintf("error checking po_approver claim: %v", err),
			}
		}
		if hasPoApproverClaim {
			// ClaimHasDivisionPermission returns a validation function that checks if the
			// provided user ID has permission for the specified division with the given claim.
			// We pass the auth.Id as the value to validate.
			if err := utilities.ClaimHasDivisionPermission(txApp, "po_approver", po.GetString("division"))(auth.Id); err == nil {
				callerIsApprover = true
			}
		}
	}

	// Check if the record requires second approval
	secondApproverClaim := po.GetString("second_approver_claim")
	recordRequiresSecondApproval := secondApproverClaim != ""

	// Check if the caller is a qualified second approver. This means the caller
	// has a user_claim with a claim_id that matches the second_approver_claim
	// on the record. Note that this is a different claim than the po_approver
	// claim and that the caller may be a second approver regardless of whether
	// they are an approver on the record.
	callerIsQualifiedSecondApprover := false // initialize to false
	if recordRequiresSecondApproval {
		userClaims, err := txApp.FindRecordsByFilter("user_claims", "uid = {:userId}", "", 0, 0, dbx.Params{
			"userId": auth.Id,
		})
		if err != nil {
			return false, false, &CodeError{
				Code:    "error_fetching_user_claims",
				Message: fmt.Sprintf("error fetching user claims: %v", err),
			}
		}
		for _, userClaim := range userClaims {
			if userClaim.GetString("cid") == secondApproverClaim {
				callerIsQualifiedSecondApprover = true
				break
			}
		}
	}

	return callerIsApprover, callerIsQualifiedSecondApprover, nil
}

// GetApprovers returns a list of users who can approve a purchase order of the given amount and division.
// If the current user has approver claims, an empty list is returned (UI will auto-set to self).
// Results are filtered to approvers with permission for the specified division
// (empty payload means all divisions, otherwise division must be in payload).
func createGetApproversHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth

		// Get the division and amount parameters
		division := e.Request.PathValue("division")
		amountStr := e.Request.PathValue("amount")
		_, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"code":    "invalid_amount",
				"message": "Amount must be a valid number",
			})
		}

		// Check if the current user has po_approver claim
		hasApproverClaim, err := utilities.HasClaim(app, auth, "po_approver")
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_checking_claims",
				"message": fmt.Sprintf("Error checking user claims: %v", err),
			})
		}

		// If user has approver claim, return empty list (UI will auto-set to self)
		if hasApproverClaim {
			return e.JSON(http.StatusOK, []map[string]string{})
		}

		// Find the po_approver claim ID
		approverClaim, err := app.FindFirstRecordByFilter("claims", "name = {:claimName}", dbx.Params{
			"claimName": "po_approver",
		})
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_claim",
				"message": fmt.Sprintf("Error fetching po_approver claim: %v", err),
			})
		}

		// Build query to get all users with po_approver claim
		query := app.DB().Select("p.uid AS id, p.given_name, p.surname").
			From("profiles p").
			InnerJoin("user_claims u", dbx.NewExp("p.uid = u.uid")).
			Where(dbx.NewExp("u.cid = {:claimId}", dbx.Params{
				"claimId": approverClaim.Id,
			}))

		// Apply division filtering
		// Include users with empty payload (all divisions) or payload containing this division
		query = query.AndWhere(dbx.NewExp("(u.payload IS NULL OR u.payload = '[]' OR u.payload = '{}' OR JSON_EXTRACT(u.payload, '$') LIKE {:divisionPattern})", dbx.Params{
			"divisionPattern": "%\"" + division + "\"%",
		}))

		// Execute the query
		var results []map[string]string
		err = query.All(&results)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_approvers",
				"message": fmt.Sprintf("Error fetching approvers: %v", err),
			})
		}

		return e.JSON(http.StatusOK, results)
	}
}

// GetSecondApprovers returns a list of users who can provide second approval for a purchase order
// of the given amount and division. If the amount is below tier 1 or the current user has the appropriate claim
// for the required tier, an empty list is returned (no second approval needed or UI will auto-set to self).
// Results are filtered to approvers with permission for the specified division
// (empty payload means all divisions, otherwise division must be in payload).
func createGetSecondApproversHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth

		// Get the division and amount parameters
		division := e.Request.PathValue("division")
		amountStr := e.Request.PathValue("amount")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"code":    "invalid_amount",
				"message": "Amount must be a valid number",
			})
		}

		// Find the appropriate claim for this amount
		secondApproverClaimId, err := utilities.FindTierForAmount(app, amount)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_determining_tier",
				"message": fmt.Sprintf("Error determining approval tier: %v", err),
			})
		}

		// If no second approval is needed (amount below tier 1), return empty list
		if secondApproverClaimId == "" {
			return e.JSON(http.StatusOK, []map[string]string{})
		}

		// Get the claim details to check if the current user has this claim
		secondApproverClaim, err := app.FindRecordById("claims", secondApproverClaimId)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_claim",
				"message": fmt.Sprintf("Error fetching claim details: %v", err),
			})
		}

		// Check if the current user has the required claim
		hasRequiredClaim, err := utilities.HasClaim(app, auth, secondApproverClaim.GetString("name"))
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_checking_claims",
				"message": fmt.Sprintf("Error checking user claims: %v", err),
			})
		}

		// If user has the required claim, return empty list (UI will auto-set to self)
		if hasRequiredClaim {
			return e.JSON(http.StatusOK, []map[string]string{})
		}

		// Build query to get all users with the required claim
		query := app.DB().Select("p.uid AS id, p.given_name, p.surname").
			From("profiles p").
			InnerJoin("user_claims u", dbx.NewExp("p.uid = u.uid")).
			Where(dbx.NewExp("u.cid = {:claimId}", dbx.Params{
				"claimId": secondApproverClaimId,
			}))

		// Apply division filtering
		// Include users with empty payload (all divisions) or payload containing this division
		query = query.AndWhere(dbx.NewExp("(u.payload IS NULL OR u.payload = '[]' OR u.payload = '{}' OR JSON_EXTRACT(u.payload, '$') LIKE {:divisionPattern})", dbx.Params{
			"divisionPattern": "%\"" + division + "\"%",
		}))

		// Execute the query
		var results []map[string]string
		err = query.All(&results)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_approvers",
				"message": fmt.Sprintf("Error fetching approvers: %v", err),
			})
		}

		return e.JSON(http.StatusOK, results)
	}
}
