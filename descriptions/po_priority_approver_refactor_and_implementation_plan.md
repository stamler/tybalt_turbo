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

## Refactoring Strategy

### 1. Create a Core Utility Function

Create a utility function that handles the common logic of finding approvers with parameters to control behavior:

```go
// GetApproversByTier fetches a list of users who can approve a purchase order based on parameters
// Parameters:
// - app: the application context
// - auth: the authenticated user record
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

### 3. Hook Validation Implementation

Create a validation function in the purchase orders hooks to validate the priority_second_approver field:

```go
// ValidatePrioritySecondApprover checks if the provided priority_second_approver ID
// is valid for the given purchase order amount and division
func ValidatePrioritySecondApprover(app core.App, record *core.Record) error {
    // Get necessary values from the record
    prioritySecondApprover := record.GetString("priority_second_approver")
    
    // If empty, validation passes (field is optional)
    if prioritySecondApprover == "" {
        return nil
    }
    
    // Get purchase order amount and division
    division := record.GetString("division")
    amount := CalculateTotalPOValue(record)
    
    // Use the utility function to get eligible second approvers
    approvers, _, err := GetApproversByTier(app, nil, division, amount, true)
    if err != nil {
        return err
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
            Status: http.StatusBadRequest,
            Message: "Invalid priority second approver",
            Data: map[string]CodeError{
                "priority_second_approver": {
                    Code:    "invalid_priority_second_approver",
                    Message: "The selected priority second approver is not authorized to approve this purchase order",
                },
            },
        }
    }
    
    return nil
}
```

## Implementation Steps

1. Create the `GetApproversByTier` utility function in `app/utilities/approvers.go`
2. Update the existing handler functions to use this utility function
3. Add validation for the priority_second_approver field in the purchase orders hook
4. Add tests for the validation logic

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

### Step 3: Add Validation Logic

Update the BeforeCreateRequest and BeforeUpdateRequest hooks in `app/hooks/purchase_orders.go` to validate the priority_second_approver field.

### Step 4: Write Tests

Create tests for:

- The utility function
- The validation logic

## Benefits of This Approach

1. **Reduces Code Duplication**: Common logic is moved to a single utility function
2. **Improves Maintainability**: Changes to approval logic only need to be made in one place
3. **Enables Reuse**: The same function can be used for API responses and internal validation
4. **Enforces Business Rules**: Ensures that only eligible second approvers can be assigned to a PO

## Considerations

- Make sure to handle the case where no second approval is needed (amount below tier1) appropriately
- Ensure proper error handling throughout the refactored code
- Consider performance implications of executing the same query twice (once for the API endpoint, once for validation)
