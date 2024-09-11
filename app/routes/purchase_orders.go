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

func createPurchaseOrderHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req PurchaseOrderRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var transactionError error
		var httpResponseStatusCode int

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			// Validate input
			if err := validatePurchaseOrderRequest(&req); err != nil {
				transactionError = err
				httpResponseStatusCode = http.StatusBadRequest
				return err
			}

			// Get the user's manager
			profile, err := txDao.FindFirstRecordByFilter("profiles", "uid = {:userId}", dbx.Params{
				"userId": userId,
			})
			if err != nil {
				return fmt.Errorf("error fetching user profile: %v", err)
			}
			approver := profile.Get("manager")

			// Determine second approver claim
			secondApproverClaim, err := getSecondApproverClaim(txDao, req.Type, req.Total)
			if err != nil {
				return err
			}

			// Create new purchase order
			poCollection, err := app.Dao().FindCollectionByNameOrId("purchase_orders")
			if err != nil {
				return fmt.Errorf("error fetching purchase_orders collection: %v", err)
			}

			newPO := models.NewRecord(poCollection)
			newPO.Set("uid", userId)
			newPO.Set("status", "Unapproved")
			newPO.Set("type", req.Type)
			newPO.Set("date", req.Date)
			newPO.Set("end_date", req.EndDate)
			newPO.Set("frequency", req.Frequency)
			newPO.Set("division", req.Division)
			newPO.Set("description", req.Description)
			newPO.Set("total", req.Total)
			newPO.Set("payment_type", req.PaymentType)
			newPO.Set("vendor_name", req.VendorName)
			newPO.Set("approver", approver)
			newPO.Set("second_approver_claim", secondApproverClaim)

			// Handle file attachment
			// file, err := c.FormFile("attachment")
			// if err == nil {
			// 	// Process file upload
			// 	if err := handleFileUpload(newPO, file); err != nil {
			// 		return err
			// 	}
			// }

			if err := txDao.SaveRecord(newPO); err != nil {
				return fmt.Errorf("error creating new purchase order: %v", err)
			}

			return nil
		})

		if err != nil {
			if transactionError != nil {
				return c.JSON(httpResponseStatusCode, map[string]string{"error": transactionError.Error()})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Purchase order created successfully"})
	}
}

func updatePurchaseOrderHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.PathParam("id")
		var req PurchaseOrderRequest
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

			// Check if the user is the creator of the purchase order
			if po.Get("uid") != userId {
				transactionError = fmt.Errorf("you are not authorized to update this purchase order")
				httpResponseStatusCode = http.StatusForbidden
				return transactionError
			}

			// Check if the purchase order is unapproved
			if po.Get("status") != "Unapproved" {
				transactionError = fmt.Errorf("only unapproved purchase orders can be updated")
				httpResponseStatusCode = http.StatusBadRequest
				return transactionError
			}

			// Validate input
			if err := validatePurchaseOrderRequest(&req); err != nil {
				transactionError = err
				httpResponseStatusCode = http.StatusBadRequest
				return err
			}

			// Determine second approver claim
			secondApproverClaim, err := getSecondApproverClaim(txDao, req.Type, req.Total)
			if err != nil {
				return err
			}

			// Update purchase order
			po.Set("type", req.Type)
			po.Set("date", req.Date)
			po.Set("end_date", req.EndDate)
			po.Set("frequency", req.Frequency)
			po.Set("division", req.Division)
			po.Set("description", req.Description)
			po.Set("total", req.Total)
			po.Set("payment_type", req.PaymentType)
			po.Set("vendor_name", req.VendorName)
			po.Set("second_approver_claim", secondApproverClaim)

			// Handle file attachment
			// file, err := c.FormFile("attachment")
			// if err == nil {
			// 	// Process file upload
			// 	if err := handleFileUpload(po, file); err != nil {
			// 		return err
			// 	}
			// }

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

		return c.JSON(http.StatusOK, map[string]string{"message": "Purchase order updated successfully"})
	}
}

func deletePurchaseOrderHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
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

			// Check if the user is the creator of the purchase order
			if po.Get("uid") != userId {
				transactionError = fmt.Errorf("you are not authorized to delete this purchase order")
				httpResponseStatusCode = http.StatusForbidden
				return transactionError
			}

			// Check if the purchase order is unapproved
			if po.Get("status") != "Unapproved" {
				transactionError = fmt.Errorf("only unapproved purchase orders can be deleted")
				httpResponseStatusCode = http.StatusBadRequest
				return transactionError
			}

			// Delete the purchase order
			if err := txDao.DeleteRecord(po); err != nil {
				return fmt.Errorf("error deleting purchase order: %v", err)
			}

			return nil
		})

		if err != nil {
			if transactionError != nil {
				return c.JSON(httpResponseStatusCode, map[string]string{"error": transactionError.Error()})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Purchase order deleted successfully"})
	}
}

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

func validatePurchaseOrderRequest(input *PurchaseOrderRequest) error {

	// Validate recurring-specific fields
	if input.Type == "Recurring" {
		if input.EndDate == "" {
			return fmt.Errorf("end_date is required for recurring purchase orders")
		}
		// verify that the end_date is a date in the format "2006-01-02"
		_, err := time.Parse("2006-01-02", input.EndDate)
		if err != nil {
			return fmt.Errorf("invalid end_date format: must be YYYY-MM-DD")
		}
		if input.Frequency == "" {
			return fmt.Errorf("frequency is required for recurring purchase orders")
		}
	}

	return nil
}

func getSecondApproverClaim(txDao *daos.Dao, poType string, total float64) (string, error) {
	var secondApproverClaim string

	// Check if the purchase order is recurring or if the total is greater than or equal to VP_PO_LIMIT
	if poType == "Recurring" || total >= VP_PO_LIMIT {
		// Set second approver claim to 'smg'
		claim, err := txDao.FindFirstRecordByFilter("claims", "name = {:claimName}", dbx.Params{
			"claimName": "smg",
		})
		if err != nil {
			return "", fmt.Errorf("error fetching SMG claim: %v", err)
		}
		secondApproverClaim = claim.Id
	} else if total >= MANAGER_PO_LIMIT {
		// Set second approver claim to 'vp'
		claim, err := txDao.FindFirstRecordByFilter("claims", "name = {:claimName}", dbx.Params{
			"claimName": "vp",
		})
		if err != nil {
			return "", fmt.Errorf("error fetching VP claim: %v", err)
		}
		secondApproverClaim = claim.Id
	}

	// If neither condition is met, secondApproverClaim remains an empty string

	return secondApproverClaim, nil
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

// func handleFileUpload(record *models.Record, file *multipart.FileHeader) error {
// 	// Open the uploaded file
// 	src, err := file.Open()
// 	if err != nil {
// 		return fmt.Errorf("error opening uploaded file: %v", err)
// 	}
// 	defer src.Close()

// 	// Create a temporary file
// 	tempFile, err := os.CreateTemp("", "upload-*.tmp")
// 	if err != nil {
// 		return fmt.Errorf("error creating temporary file: %v", err)
// 	}
// 	defer tempFile.Close()
// 	defer os.Remove(tempFile.Name())

// 	// Copy the uploaded file to the temporary file
// 	_, err = io.Copy(tempFile, src)
// 	if err != nil {
// 		return fmt.Errorf("error copying uploaded file: %v", err)
// 	}

// 	// Seek to the beginning of the file
// 	_, err = tempFile.Seek(0, 0)
// 	if err != nil {
// 		return fmt.Errorf("error seeking file: %v", err)
// 	}

// 	// Set the file to the record
// 	if err := record.SetDataFile("attachment", file.Filename, tempFile); err != nil {
// 		return fmt.Errorf("error setting file to record: %v", err)
// 	}

// 	return nil
// }
