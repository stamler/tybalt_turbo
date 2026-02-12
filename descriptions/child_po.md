# Child Purchase Order System

## Overview

When a Cumulative PO's total is exceeded by an expense, rather than simply
returning an error, a provision exists to create a child PO that inherits
properties from the parent PO. This streamlines the process of handling overflow
expenses while maintaining proper tracking and relationships.

## Implementation Plan

### 1. Database Schema Updates

1. Add new fields to `purchase_orders` collection:
   - `parent_po` (relation) - References the parent purchase_orders record (required for child POs)
   - fail when creating a child PO with a type other than `One-Time`
   - fail when creating a child PO of an existing child PO (child POs cannot be parents)
   - Update SQL pattern matching index for new PO number format

### 2. Backend Changes

#### A. Purchase Order Hook Enhancement

1. Modify `cleanPurchaseOrder` function in `app/hooks/purchase_orders.go`:
     - Handle second_approver_claim based on total value

2. Modify `ProcessPurchaseOrder` function:
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

#### C. PO Number Generation Enhancement

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

1. Create new component `ChildPoCreationDialog.svelte`:
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

2. Integration Tests:
   - Test complete workflow from expense submission to child PO creation
   - Test approval process for child POs
   - Test PO number generation timing
   - Test parent PO constraints:
     - Cannot be cancelled with children
     - Cannot change approval status with children
   - Test multiple overflow scenarios
   - Test concurrent child PO creation attempts

### 5. Documentation Updates

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
   - Child POs must be One-Time type
   - No multiple children for same expense
   - Parent PO protection rules
   - Approval workflow integrity

## Limitations and Considerations

1. Consider limits on number of child POs per parent (max 99 with XX format)
2. Handle edge cases like:
   - Cancellation of parent PO (we should prevent cancellation of parent PO if it has child POs)
   - Multiple simultaneous overflow attempts. We should prevent multiple overflow attempts from the same expense by denying the request if the parent PO already has a child PO.
   - Approval order dependencies
   - Parent PO approval status changes. We should prevent changes to the parent PO's approval status if it has child POs.
