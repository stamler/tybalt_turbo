package routes

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	MANAGER_PO_LIMIT = 500
	VP_PO_LIMIT      = 2500
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
					Code:    "po_already_rejected",
					Message: "this purchase order has been rejected and cannot be approved",
				}
			}

			/*
				The caller may not be the approver but still be qualified to approve
				the purchase order if they have a po_approver claim and the payload
				specifies a division that matches the record's division. In this case,
				callerIsQualifiedApprover is set to true and then during the update we
				will set approver to the caller's uid.
			*/

			// Check if the user is the approver, a qualified approver, or a qualified
			// second approver
			callerIsApprover, callerIsQualifiedApprover, callerIsQualifiedSecondApprover, err := isApprover(txApp, userId, po)
			if err != nil {
				return err
			}
			recordIsApproved := !po.GetDateTime("approved").IsZero()
			recordRequiresSecondApproval := po.GetString("second_approver_claim") != ""
			recordIsSecondApproved := !po.GetDateTime("second_approval").IsZero()

			// This time will be written to the record if the approval or second
			// approval status changes
			now := time.Now()

			if recordIsApproved && callerIsApprover && recordRequiresSecondApproval && !recordIsSecondApproved && !callerIsQualifiedSecondApprover {
				// if the caller is the approver and approved is already set and the
				// record requires second approval but the caller is not qualified to
				// approve it, return an error indicating that the purchase order has
				// already been approved by a manager but requires elevated approval.
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "po_missing_second_approval",
					Message: "this purchase order has already been approved by a manager but requires second approval",
				}
			} else if recordIsApproved && callerIsQualifiedSecondApprover && recordRequiresSecondApproval && !recordIsSecondApproved {
				// Second-Approve the purchase order
				po.Set("second_approval", now)
				po.Set("second_approver", userId)
				recordIsSecondApproved = true
			} else if !recordIsApproved {
				if callerIsApprover {
					// Approve the purchase order as is
					po.Set("approved", now)
					recordIsApproved = true
				} else if callerIsQualifiedApprover {
					// Approve the purchase order, updating the approver to the caller's
					// uid since the caller is not the approver specified in the record
					po.Set("approved", now)
					po.Set("approver", userId)
					recordIsApproved = true
				}
			} else {
				// the user is not the approver or a qualified second approver
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized_approval",
					Message: "you are not authorized to approve this purchase order",
				}
			}

			// If approved is set and either second_approver_claim is not set or
			// second_approval is set, the purchase order is fully approved. Set the
			// status to Active and generate a PO number
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

		var req RejectionRequest
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "you must provide a rejection reason",
				"code":    "invalid_request_body",
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
					Code:    "po_already_rejected",
					Message: "this purchase order has been rejected and cannot be rejected again",
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

			// Check if the user is the approver or a qualified second approver
			isApprover := po.Get("approver") == userId
			isSecondApprover := false

			if !isApprover {
				secondApproverClaim := po.Get("second_approver_claim")
				if secondApproverClaim != nil {
					userClaims, err := txApp.FindRecordsByFilter("user_claims", "uid = {:userId}", "", 0, 0, dbx.Params{
						"userId": userId,
					})
					if err != nil {
						httpResponseStatusCode = http.StatusInternalServerError
						return &CodeError{
							Code:    "error_fetching_user_claims",
							Message: fmt.Sprintf("error fetching user claims: %v", err),
						}
					}
					for _, claim := range userClaims {
						if claim.Get("claim") == secondApproverClaim {
							isSecondApprover = true
							break
						}
					}
				}
			}

			if !isApprover && !isSecondApprover {
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

		return e.JSON(http.StatusOK, map[string]string{"message": "Purchase order rejected successfully"})
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
			hasPayablesAdminClaim, err := utilities.HasClaim(txApp, userId, "payables_admin")
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
		return e.NoContent(http.StatusNoContent)
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
			hasPayablesAdminClaim, err := utilities.HasClaim(txApp, userId, "payables_admin")
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

		return e.NoContent(http.StatusNoContent)
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
	checks and returns 3 boolean values indicating:
		1. whether the caller is the approver specified in the record
		2. whether the caller is a qualified approver for another reason as outlined above
		3. whether the caller is a qualified second approver
	We will incorporate this function into the approval logic above within the
	createApprovePurchaseOrderHandler function.
*/

func isApprover(txApp core.App, userId string, po *core.Record) (bool, bool, bool, error) {
	// Check if the caller is the approver specified in the record
	callerIsApprover := po.GetString("approver") == userId
	callerIsQualifiedApprover := false

	// if the caller is not the approver, perform additional checks to see if
	// caller is a qualified approver
	if !callerIsApprover {

		// Check if the caller is a qualified approver (has the po_approver claim
		// and that claim has a payload that includes the division of the purchase
		// order)
		// TODO: implement this (perhaps HasClaim should also return the payload?)
		callerIsQualifiedApprover = false
		hasPoApproverClaim, err := utilities.HasClaim(txApp, userId, "po_approver")
		if err != nil {
			return false, false, false, &CodeError{
				Code:    "error_checking_po_approver_claim",
				Message: fmt.Sprintf("error checking po_approver claim: %v", err),
			}
		}
		if hasPoApproverClaim {
			// TODO: check if the payload includes the division of the purchase order
			callerIsQualifiedApprover = true
		}
	}

	// Check if the record requires second approval
	secondApproverClaim := po.GetString("second_approver_claim")
	recordRequiresSecondApproval := secondApproverClaim != ""

	// Check if the caller is a qualified second approver
	callerIsQualifiedSecondApprover := false // initialize to false
	if recordRequiresSecondApproval {
		userClaims, err := txApp.FindRecordsByFilter("user_claims", "uid = {:userId}", "", 0, 0, dbx.Params{
			"userId": userId,
		})
		if err != nil {
			return false, false, false, &CodeError{
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

	return callerIsApprover, callerIsQualifiedApprover, callerIsQualifiedSecondApprover, nil
}
