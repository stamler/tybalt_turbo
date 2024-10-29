package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"tybalt/utilities"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

const (
	MANAGER_PO_LIMIT = 500
	VP_PO_LIMIT      = 2500
)

func createApprovePurchaseOrderHandler(app core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.PathParam("id")

		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			// Fetch existing purchase order
			po, err := txDao.FindRecordById("purchase_orders", id)
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
			callerIsApprover, callerIsQualifiedApprover, callerIsQualifiedSecondApprover, err := isApprover(txDao, userId, po)
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
				poNumber, err := generatePONumber(txDao)
				if err != nil {
					return &CodeError{
						Code:    "error_generating_po_number",
						Message: fmt.Sprintf("error generating PO number: %v", err),
					}
				}
				po.Set("po_number", poNumber)
			}

			if err := txDao.SaveRecord(po); err != nil {
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
				return c.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// return the updated purchase order from the database
		updatedPO, err := app.Dao().FindRecordById("purchase_orders", id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Expand the new purchase order
		if errs := app.Dao().ExpandRecord(updatedPO, []string{"uid.profiles_via_uid", "approver.profiles_via_uid", "division", "job"}, nil); len(errs) > 0 {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"message": fmt.Sprintf("error expanding record: %v", errs),
				"code":    "error_expanding_record",
			})
		}
		// return the updated purchase order as a JSON response
		return c.JSON(http.StatusOK, updatedPO)
	}
}

func createRejectPurchaseOrderHandler(app core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.PathParam("id")

		var req RejectionRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "you must provide a rejection reason",
				"code":    "invalid_request_body",
			})
		}

		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			// Fetch existing purchase order
			po, err := txDao.FindRecordById("purchase_orders", id)
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
					userClaims, err := txDao.FindRecordsByFilter("user_claims", "uid = {:userId}", "", 0, 0, dbx.Params{
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

			if err := txDao.SaveRecord(po); err != nil {
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
				return c.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Purchase order rejected successfully"})
	}
}

func createCancelPurchaseOrderHandler(app core.App) echo.HandlerFunc {
	return func(c echo.Context) error {
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var httpResponseStatusCode int
		id := c.PathParam("id")

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			// Fetch existing purchase order
			po, err := txDao.FindRecordById("purchase_orders", id)
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
			hasAccountingClaim, err := utilities.HasClaim(txDao, userId, "accounting")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_user_claims",
					Message: fmt.Sprintf("error fetching user claims: %v", err),
				}
			}
			if !hasAccountingClaim {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized_cancellation",
					Message: "you are not authorized to cancel this purchase order",
				}
			}

			// Count the number of associated expenses records. If there are any, the
			// purchase order cannot be cancelled.
			expenses, err := txDao.FindRecordsByFilter("expenses", "purchase_order = {:poId}", "", 0, 0, dbx.Params{
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

			if err := txDao.SaveRecord(po); err != nil {
				return &CodeError{
					Code:    "error_updating_purchase_order",
					Message: fmt.Sprintf("error updating purchase order: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			if codeError, ok := err.(*CodeError); ok {
				return c.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Purchase order cancelled successfully"})
	}
}

func generatePONumber(txDao *daos.Dao) (string, error) {
	currentYear := time.Now().Year()
	prefix := fmt.Sprintf("%d-", currentYear)

	// Query existing PO numbers for the current year
	existingPOs, err := txDao.FindRecordsByFilter(
		"purchase_orders",
		"po_number ~ {:prefix}",
		"-po_number",
		1,
		0,
		dbx.Params{"prefix": prefix},
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
		existing, err := txDao.FindFirstRecordByFilter(
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
	the isApprover function with 3 arguments: the txDao, the userId of the
	caller, and the purchase_orders record. The function performs the following
	checks and returns 3 boolean values indicating:
		1. whether the caller is the approver specified in the record
		2. whether the caller is a qualified approver for another reason as outlined above
		3. whether the caller is a qualified second approver
	We will incorporate this function into the approval logic above within the
	createApprovePurchaseOrderHandler function.
*/

func isApprover(txDao *daos.Dao, userId string, po *models.Record) (bool, bool, bool, error) {
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
		hasPoApproverClaim, err := utilities.HasClaim(txDao, userId, "po_approver")
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
		userClaims, err := txDao.FindRecordsByFilter("user_claims", "uid = {:userId}", "", 0, 0, dbx.Params{
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
