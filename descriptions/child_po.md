# Child Purchase Order Feature Implementation Plan

## Overview

When a Cumulative PO's total is exceeded by an expense, instead of just returning an error, we'll provide functionality to create a child PO that inherits properties from the parent PO. This will streamline the process of handling overflow expenses while maintaining proper tracking and relationships.

## Current System Analysis

1. The system currently:
   - Validates expenses against PO totals with a maximum excess limit
   - Has specific handling for Cumulative POs in expense validation
   - Generates PO numbers only after full approval using format `YYYY-NNNN`
   - Uses the backend `generatePONumber()` function with SQL pattern matching
   - Validates PO types (Normal, Cumulative, Recurring) with specific rules for each
   - Prevents changes to POs after approval
   - Uses a two-phase validation approach (clean then validate)
   - Handles all operations within transactions
   - Has a sophisticated approval process with multiple checks

## Implementation Plan

### 1. Database Schema Updates

1. Add new fields to `purchase_orders` collection:
   - `parent_po` (relation) - References the parent purchase_orders record (required for child POs)
   - Add validation rule to ensure child POs can only be of type `Normal` (this should already done by the validation rule that ensures child POs can only be of type `Normal`)
   - Add validation rule to prevent child POs from having their own children (this should already done by the validation rule that ensures child POs can only be of type `Normal`)
   - Update SQL pattern matching index for new PO number format

### 2. Backend Changes

#### A. Purchase Order Hook Enhancement

1. Modify `cleanPurchaseOrder` function in `app/hooks/purchase_orders.go`:
   - Add child PO cleaning logic similar to Normal/Cumulative handling:
     - Clear end_date and frequency for child POs
     - Set type to Normal for child POs
     - Handle second_approver_claim based on total value

2. Modify `validatePurchaseOrder` function:
   - Add parent-child validation rules:

     ```go
     "parent_po": validation.Validate(
         purchaseOrderRecord.Get("parent_po"),
         validation.When(isChild,
             validation.Required.Error("parent_po is required for child POs"),
             validation.By(parentMustBeCumulative),
             validation.By(parentMustBeActive),
         ).Else(
             validation.Empty.Error("parent_po must be empty for non-child POs"),
         ),
     ),
     "type": validation.Validate(
         purchaseOrderRecord.Get("type"),
         validation.When(isChild,
             validation.In("Normal").Error("child POs must be Normal type"),
         ),
     ),
     ```

3. Modify `ProcessPurchaseOrder` function:
   - Add transaction-wrapped validation for parent-child relationships
   - Add checks for parent PO status changes
   - Add validation for concurrent child PO creation

#### B. Expense Validation Hook Enhancement

1. Modify `validateExpense` function in `app/hooks/validate_expenses.go`:
   - This occurs when POSTing to the /api/collections/expenses/records endpoint
   - When detecting a Cumulative PO excess return a special error code indicating overflow possibility
   - Include parent PO details in the error response
   - Use the following as psuedo code for the logic. It may look like it's not complete but it's just a starting point.

     ```go
     if hasPurchaseOrder && (poType == "Cumulative") {
         // Check for existing child PO within transaction
         existingChild, err := txDao.FindFirstRecordByFilter(
             "purchase_orders",
             "parent_po = {:parentId} AND status != 'Cancelled'",
             dbx.Params{"parentId": poRecord.Id},
         )
         if err != nil && err != sql.ErrNoRows {
             return err
         }
         if existingChild != nil {
             return &CodeError{
                 Code: "existing_child_po",
                 Message: "parent PO already has a child PO",
                 Data: map[string]interface{}{
                     "child_po": existingChild.Id,
                 },
             }
         }
         // Return special overflow error with details
         return &CodeError{
             Code: "cumulative_po_overflow",
             Message: fmt.Sprintf("cumulative expenses exceed purchase order total of $%0.2f by $%0.2f", poTotal, overflowAmount),
             Data: map[string]interface{}{
                 "parent_po": poRecord,
                 "overflow_amount": overflowAmount,
             },
         }
     }
     ```

#### C. New API Endpoint

1. Create `/api/purchase_orders/:id/create_child` endpoint:
   - Input: Original expense details that caused overflow
   - Functionality:
     - Creates new PO with inherited properties
     - Sets required `parent_po` reference
     - Pre-fills remaining fields from parent
     - Calculates appropriate total based on overflow amount
     - Leaves `po_number` blank (will be assigned upon approval)

2. Use the following as psuedo code for the logic. It may look like it's not complete but it's just a starting point.

   ```go
   func createChildPurchaseOrderHandler(app core.App) echo.HandlerFunc {
       return func(c echo.Context) error {
           return app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
               // Validate parent PO
               parentId := c.PathParam("id")
               parentPO, err := txDao.FindRecordById("purchase_orders", parentId)
               if err != nil {
                   return err
               }
               
               // Validation checks
               if parentPO.GetString("type") != "Cumulative" {
                   return &CodeError{
                       Code: "invalid_parent_type",
                       Message: "parent must be Cumulative type",
                   }
               }
               // ... more validation ...
               
               // Create child PO
               childPO := models.NewRecord("purchase_orders")
               childPO.Set("parent_po", parentId)
               childPO.Set("type", "Normal")
               // ... inherit other properties ...
               
               return txDao.SaveRecord(childPO)
           })
       }
   }
   ```

#### D. PO Number Generation Enhancement

1. Modify `generatePONumber()` function:
   - Check for presence of `parent_po` field
   - If `parent_po` is not set:
     - Use existing format: `YYYY-NNNN`
   - If `parent_po` is set:
     - Retrieve parent's PO number (format: `YYYY-NNNN`)
     - Query for existing children to determine next available suffix
     - Generate child number with format: `YYYY-NNNN-XX`
     - Where XX is zero-padded sequential number (01, 02, etc.)

2. Update unique index constraints to handle new format

3. Use the following as psuedo code for the logic. It may look like it's not complete but it's just a starting point.

   ```go
   func generatePONumber(txDao *daos.Dao, record *models.Record) (string, error) {
       currentYear := time.Now().Year()
       prefix := fmt.Sprintf("%d-", currentYear)
       
       // If this is a child PO, handle differently
       if record.Get("parent_po") != "" {
           parent, err := txDao.FindRecordById("purchase_orders", record.Get("parent_po").(string))
           if err != nil {
               return "", err
           }
           parentNumber := parent.GetString("po_number")
           
           // Find highest child suffix
           children, err := txDao.FindRecordsByFilter(
               "purchase_orders",
               "po_number ~ {:pattern}",
               "-po_number",
               1,
               0,
               dbx.Params{"pattern": fmt.Sprintf("^%s-\\d{2}$", parentNumber)},
           )
           
           var nextSuffix int = 1
           if len(children) > 0 {
               lastChild := children[0].GetString("po_number")
               fmt.Sscanf(lastChild[len(lastChild)-2:], "%d", &nextSuffix)
               nextSuffix++
           }
           
           if nextSuffix > 99 {
               return "", fmt.Errorf("maximum number of child POs reached")
           }
           
           return fmt.Sprintf("%s-%02d", parentNumber, nextSuffix), nil
       }
       
       // Original logic for parent POs
       // ... existing code for YYYY-NNNN format ...
   }
   ```

#### E. Utilities

1. New utility functions in `app/utilities/purchase_orders.go`:

   ```go
   // CalculateOverflowAmount calculates the excess amount for a child PO
   func CalculateOverflowAmount(expense *models.Record, parentPO *models.Record, existingTotal float64) float64 {
       // ... implementation ...
   }
   
   // ValidateChildPo validates child PO constraints
   func ValidateChildPo(txDao *daos.Dao, childPO *models.Record) error {
       // ... implementation ...
   }
   
   // HasExistingChildPo checks for existing child POs
   func HasExistingChildPo(txDao *daos.Dao, parentId string) (bool, error) {
       // ... implementation ...
   }
   ```

### 3. Frontend Changes

#### A. UI Components

1. Enhance error handling in `ExpensesEditor.svelte`:
   - Add handling for new `cumulative_po_overflow` error type
   - Show modal/dialog when overflow detected
   - Provide options:
     - Create child PO
     - Modify expense amount
     - Cancel

2. Create new component `ChildPoCreationDialog.svelte`:
   - Display parent PO details
   - Show overflow amount
   - Allow adjustment of new PO total
   - Display placeholder for future PO number format
   - Submit button to create child PO
   - Display validation errors

### 4. Testing Plan

1. Unit Tests:
   - Test PO number generation for both parent and child formats
   - Test suffix generation sequence (01, 02, etc.)
   - Validate inheritance of properties
   - Test overflow amount calculations
   - Verify proper linking of parent-child relationships
   - Verify `parent_po` field requirements

2. Perhaps this pseudo code could be helpful for the unit tests:

   ```go
   func TestGeneratePONumber(t *testing.T) {
       tests := []struct {
           name     string
           record   *models.Record
           want     string
           wantErr  bool
       }{
           {
               name: "normal parent PO",
               record: &models.Record{...},
               want: "2024-0001",
               wantErr: false,
           },
           {
               name: "first child PO",
               record: &models.Record{
                   // parent with number 2024-0001
               },
               want: "2024-0001-01",
               wantErr: false,
           },
           // ... more test cases ...
       }
       // ... test implementation ...
   }
   ```

3. Integration Tests:
   - Test complete workflow from expense submission to child PO creation
   - Test approval process for child POs
   - Test PO number generation timing
   - Test parent PO constraints:
     - Cannot be cancelled with children
     - Cannot change approval status with children
   - Test multiple overflow scenarios
   - Test concurrent child PO creation attempts

### 5. Migration Strategy

1. Database Migration will be done manually since it uses PocketBase migrations:
   - Add `parent_po` field with appropriate constraints. Specifically, it will be a relation to the `purchase_orders` collection and is not required for non-child POs.
   - Update indexes to handle new PO number format. This will be done by dropping the existing index and creating a new one with the new pattern.
   - Add validation for parent-child relationships. Specifically, it will be a validation rule that ensures that child POs can only be of type `Normal` and that they cannot have their own children.

2. Deployment Steps:
   - Deploy database changes first
   - Deploy backend changes with feature flag
   - Deploy frontend changes
   - Enable feature flag in production

### 6. Documentation Updates

1. Update API documentation:
   - Document new endpoint
   - Describe both PO number formats:
     - Parent format: `YYYY-NNNN`
     - Child format: `YYYY-NNNN-XX`
   - Document new error codes
   - Document `parent_po` field requirements
   - Document constraints on parent and child POs

2. Update user documentation:
   - Explain child PO concept
   - Document workflow for handling overflows
   - Provide examples of common scenarios
   - Explain PO number assignment timing
   - Document limitations on child POs

## Success Criteria

1. Users can seamlessly create child POs when overflow occurs
2. Child POs maintain proper relationship with parent via `parent_po` field
3. PO numbering system correctly implements both formats:
   - Parent: `YYYY-NNNN`
   - Child: `YYYY-NNNN-XX`
4. PO numbers are only assigned after full approval
5. All existing PO functionality works with child POs
6. System maintains data integrity and audit trail
7. UI provides clear guidance and feedback during overflow scenarios
8. System properly enforces all constraints:
   - Child POs must be Normal type
   - No multiple children for same expense
   - Parent PO protection rules
   - Approval workflow integrity

## Limitations and Considerations

1. Child POs will require same approval process as normal POs based on their value. They can only be of type `Normal`, not `Recurring` or `Cumulative`.
2. Need to consider reporting implications for linked POs
3. Consider limits on number of child POs per parent (max 99 with XX format)
4. Handle edge cases like:
   - Cancellation of parent PO (we should prevent cancellation of parent PO if it has child POs)
   - Multiple simultaneous overflow attempts. We should prevent multiple overflow attempts from the same expense by denying the request if the parent PO already has a child PO.
   - Approval order dependencies
   - Parent PO approval status changes. We should prevent changes to the parent PO's approval status if it has child POs.
