package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

const (
	MANAGER_PO_LIMIT = 500
	VP_PO_LIMIT      = 2500
)

func approvePurchaseOrderHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.PathParam("id")

		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var transactionError error
		var httpResponseStatusCode int

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			// Fetch existing purchase order
			po, err := txDao.FindRecordById("purchase_orders", id)
			if err != nil {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Purchase order not found"})
			}

			// Check if the purchase order is unapproved
			if po.Get("status") != "Unapproved" {
				transactionError = fmt.Errorf("only unapproved purchase orders can be approved")
				httpResponseStatusCode = http.StatusBadRequest
				return transactionError
			}

			// Check if the purchase order is already rejected
			if po.Get("rejected") != nil {
				transactionError = fmt.Errorf("this purchase order has been rejected and cannot be approved")
				httpResponseStatusCode = http.StatusBadRequest
				return transactionError
			}

			// Check if the user is the approver or a qualified second approver
			isApprover := po.Get("approver") == userId
			isSecondApprover := false

			if !isApprover {
				secondApproverClaim := po.Get("second_approver_claim")
				if secondApproverClaim != nil {
					userClaims, err := txDao.FindRecordsByFilter("user_claims", "user = {:userId}", "", 0, 0, dbx.Params{
						"userId": userId,
					})
					if err != nil {
						return fmt.Errorf("error fetching user claims: %v", err)
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
				transactionError = fmt.Errorf("you are not authorized to approve this purchase order")
				httpResponseStatusCode = http.StatusForbidden
				return transactionError
			}

			// Update the purchase order
			now := time.Now()
			if isApprover {
				po.Set("approved", now)
			}
			if isSecondApprover {
				po.Set("second_approval", now)
				po.Set("second_approver", userId)
			}

			// Check if both approvals are complete
			if po.Get("approved") != nil && (po.Get("second_approver_claim") == nil || po.Get("second_approval") != nil) {
				po.Set("status", "Active")
				poNumber, err := generatePONumber(txDao)
				if err != nil {
					return fmt.Errorf("error generating PO number: %v", err)
				}
				po.Set("po_number", poNumber)
			}

			if err := txDao.SaveRecord(po); err != nil {
				return fmt.Errorf("error updating purchase order: %v", err)
			}

			return nil
		})

		if err != nil {
			if transactionError != nil {
				return c.JSON(httpResponseStatusCode, map[string]string{"error": transactionError.Error()})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Purchase order approved successfully"})
	}
}

func rejectPurchaseOrderHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.PathParam("id")

		var req struct {
			RejectionReason string `json:"rejection_reason"`
		}
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var transactionError error
		var httpResponseStatusCode int

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			// Fetch existing purchase order
			po, err := txDao.FindRecordById("purchase_orders", id)
			if err != nil {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Purchase order not found"})
			}

			// Check if the purchase order is unapproved
			if po.Get("status") != "Unapproved" {
				transactionError = fmt.Errorf("only unapproved purchase orders can be rejected")
				httpResponseStatusCode = http.StatusBadRequest
				return transactionError
			}

			// Check if the user is the approver or a qualified second approver
			isApprover := po.Get("approver") == userId
			isSecondApprover := false

			if !isApprover {
				secondApproverClaim := po.Get("second_approver_claim")
				if secondApproverClaim != nil {
					userClaims, err := txDao.FindRecordsByFilter("user_claims", "user = {:userId}", "", 0, 0, dbx.Params{
						"userId": userId,
					})
					if err != nil {
						return fmt.Errorf("error fetching user claims: %v", err)
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
				transactionError = fmt.Errorf("you are not authorized to reject this purchase order")
				httpResponseStatusCode = http.StatusForbidden
				return transactionError
			}

			// Update the purchase order
			po.Set("rejected", true)
			po.Set("rejection_reason", req.RejectionReason)
			po.Set("rejector", userId)

			if err := txDao.SaveRecord(po); err != nil {
				return fmt.Errorf("error updating purchase order: %v", err)
			}

			return nil
		})

		if err != nil {
			if transactionError != nil {
				return c.JSON(httpResponseStatusCode, map[string]string{"error": transactionError.Error()})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Purchase order rejected successfully"})
	}
}

func cancelPurchaseOrderHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	// print the app
	fmt.Println(app)
	return nil
}

func generatePONumber(txDao *daos.Dao) (string, error) {
	currentYear := time.Now().Year()
	prefix := fmt.Sprintf("%d-", currentYear)

	// Query existing PO numbers for the current year
	existingPOs, err := txDao.FindRecordsByFilter(
		"purchase_orders",
		"po_number ~ {:prefix}",
		"po_number DESC",
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
