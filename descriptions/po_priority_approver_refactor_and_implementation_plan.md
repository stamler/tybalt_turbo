# Priority Second Approver Refactoring and Implementation Plan

## Current Status Analysis

The backend implementation of the priority second approver feature has two handler functions with significant duplication:

1. `createGetApproversHandler`: Returns users who can approve a PO (first approvers)
2. `createGetSecondApproversHandler`: Returns users who can provide second approval for a PO

Both functions follow a similar pattern:

- Parse path parameters (division and amount)
- Check if the current user has relevant claims
- Return empty list if self-approval applies
- Build and execute SQL queries to fetch eligible approvers
- Filter results by division permissions
- Return formatted results

## Implementation Progress

### Completed

- The `cleanPurchaseOrder` function has already been updated to clear the `priority_second_approver` field when second approval is not needed:

  ```go
  // In cleanPurchaseOrder
  secondApproverClaimId, err := getSecondApproverClaimId(app, record)
  if err != nil {
      return err
  }
  
  // Set second_approver_claim on the record
  record.Set("second_approver_claim", secondApproverClaimId)
  
  // Clear priority_second_approver if no second approval is needed
  if secondApproverClaimId == "" {
      record.Set("priority_second_approver", "")
  }
  ```

### Remaining Work

The following aspects still need to be implemented:

## Refactoring Strategy

### 1. Create a Core Utility Function

Create a utility function that handles the common logic of finding approvers with parameters to control behavior:

```go
// GetApproversByTier fetches a list of users who can approve a purchase order based on parameters
// Parameters:
// - app: the application context
// - auth: the authenticated user record. This is optional (nil is a valid parameter) as nil enables using the function for validation where no authenticated user is available.
// - division: the division ID for which approval is needed
// - amount: the purchase order amount
// - forSecondApproval: whether we're looking for second approvers (true) or first approvers (false)
// Returns:
// - []Approver: list of eligible approvers
// - bool: whether the current user is among the eligible approvers
// - error: any error that occurred
func GetApproversByTier(
    app core.App, 
    auth *core.Record, 
    division string, 
    amount float64, 
    forSecondApproval bool,
) ([]Approver, bool, error) {
    // Implementation details...
}
```

### 2. Update Existing Handler Functions

Refactor both handler functions to use the new utility function:

```go
func createGetApproversHandler(app core.App) func(e *core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        // Parse parameters
        // Call GetApproversByTier with forSecondApproval=false
        // Return results
    }
}

func createGetSecondApproversHandler(app core.App) func(e *core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        // Parse parameters
        // Call GetApproversByTier with forSecondApproval=true
        // Return results
    }
}
```

### 3. Update ProcessPurchaseOrder for Priority Second Approver

The existing `ProcessPurchaseOrder()` function in `app/hooks/purchase_orders.go` follows a clean/validate pattern:

```go
func ProcessPurchaseOrder(app core.App, record *core.Record, authRecord *core.Record, processType string) error {
    // Stage 1: Clean up data
    if err := cleanPurchaseOrder(app, record, authRecord, processType); err != nil {
        return err
    }

    // Stage 2: Validate the purchase order
    if err := validatePurchaseOrder(app, record, authRecord, processType); err != nil {
        return err
    }

    // Additional stages...

    return nil
}
```

#### 3.1 Update `cleanPurchaseOrder` [COMPLETED]

The `cleanPurchaseOrder` function has already been updated to clear the priority_second_approver field if second approval is not needed.

#### 3.2 Update `validatePurchaseOrder` [PENDING]

We still need to update the `validatePurchaseOrder` function to validate the priority_second_approver field if set:

```go
func validatePurchaseOrder(app core.App, record *core.Record, authRecord *core.Record, processType string) error {
    // Existing validation logic...

    // Validate priority_second_approver if set
    prioritySecondApprover := record.GetString("priority_second_approver")
    if prioritySecondApprover != "" {
        // Get purchase order details
        division := record.GetString("division")
        
        // Calculate total value for the PO (same calculation as in cleanPurchaseOrder)
        poType := record.GetString("type")
        total := record.GetFloat("total")
        totalValue := total
        
        if poType == "Recurring" {
            _, totalValue, err = utilities.CalculateRecurringPurchaseOrderTotalValue(app, record)
            if err != nil {
                return err
            }
        }
        
        // Get list of eligible second approvers
        approvers, _, err := utilities.GetApproversByTier(app, nil, division, totalValue, true)
        if err != nil {
            return &HookError{
                Status:  http.StatusInternalServerError,
                Message: "hook error when checking eligible second approvers",
                Data: map[string]CodeError{
                    "global": {
                        Code:    "error_checking_approvers",
                        Message: fmt.Sprintf("error checking eligible second approvers: %v", err),
                    },
                },
            }
        }
        
        // Check if prioritySecondApprover is in the list of eligible approvers
        valid := false
        for _, approver := range approvers {
            if approver.ID == prioritySecondApprover {
                valid = true
                break
            }
        }
        
        if !valid {
            return &HookError{
                Status:  http.StatusBadRequest,
                Message: "hook error when validating priority second approver",
                Data: map[string]CodeError{
                    "priority_second_approver": {
                        Code:    "invalid_priority_second_approver",
                        Message: "The selected priority second approver is not authorized to approve this purchase order",
                    },
                },
            }
        }
    }

    return nil
}
```

## Implementation Steps

1. Create the `GetApproversByTier` utility function in `app/utilities/po_approvers.go`
2. Update the existing handler functions to use this utility function
3. Update `validatePurchaseOrder()` to validate the priority_second_approver field if set (cleanPurchaseOrder is already updated)
4. Add tests for the utility function and the updated hooks

## Detailed Implementation Plan

### Step 1: Create Utility Function

Create a new file `app/utilities/po_approvers.go` with the following functions:

```go
package utilities

import (
    "fmt"
    "github.com/pocketbase/dbx"
    "github.com/pocketbase/pocketbase/core"
)

// Approver represents a user who can approve purchase orders
type Approver struct {
    ID        string `db:"id" json:"id"`
    GivenName string `db:"given_name" json:"given_name"`
    Surname   string `db:"surname" json:"surname"`
}

// GetApproversByTier fetches a list of users who can approve a purchase order based on parameters
func GetApproversByTier(
    app core.App, 
    auth *core.Record, 
    division string, 
    amount float64, 
    forSecondApproval bool,
) ([]Approver, bool, error) {
    // Implementation...
}
```

### Step 2: Update Handler Functions

Update both `createGetApproversHandler` and `createGetSecondApproversHandler` to use the new utility function.

### Step 3: Update validatePurchaseOrder

Update the `validatePurchaseOrder` function in `app/hooks/purchase_orders.go` to validate the priority_second_approver field.

### Step 4: Write Tests

Create tests for:

- The utility function
- The updated hooks

## Benefits of This Approach

1. **Reduces Code Duplication**: Common logic is moved to a single utility function
2. **Improves Maintainability**: Changes to approval logic only need to be made in one place
3. **Enables Reuse**: The same function can be used for API responses and internal validation
4. **Enforces Business Rules**: Ensures that only eligible second approvers can be assigned to a PO
5. **Follows Existing Patterns**: Uses the established clean/validate pattern in ProcessPurchaseOrder
6. **Proper Separation of Concerns**:
   - The clean phase ensures data consistency (clearing priority_second_approver when not needed) [COMPLETED]
   - The validate phase ensures business rule compliance (validating against eligible approvers) [PENDING]

## Considerations

- Make sure to handle the case where no second approval is needed (amount below tier1) appropriately
- Ensure proper error handling throughout the refactored code
- Consider performance implications of executing the same query twice (once for the API endpoint, once for validation)
