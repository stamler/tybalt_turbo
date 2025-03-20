# Purchase Order Priority Second Approver Feature

## Overview

This document outlines the implementation of the priority second approver feature for purchase orders. This feature allows users to specify which holder of the appropriate approval tier claim should be responsible for providing the second approval on a purchase order.

## Purpose

When a purchase order requires second approval based on its value, the current system assigns the second_approver_claim but does not designate a specific user to handle that approval. This results in all users with the matching claim seeing the purchase order in their approval queue, creating unnecessary noise and potential confusion over responsibility.

The priority_second_approver field addresses this by:

1. Allowing the creator/editor to specifically designate a second approver
2. Creating a 24-hour window of exclusive review for the designated approver
3. Falling back to the standard all-users approach only if the designated approver doesn't act within the window

## Database Components

### Existing Components

- The `purchase_orders` table already includes the `priority_second_approver` field
- The `po_approval_tiers` table exists with the proper structure of claim/max_amount pairs
- The database-driven tier system is implemented, replacing hardcoded constants

### Redundant Components

- The `po_approvers` view will be redundant. This view currently:
  - Provides a static list of users with the po_approver claim
  - Is used in the UI for populating approver dropdown lists:
    - In `ui/src/routes/pos/add/+page.ts`
    - In `ui/src/routes/pos/[poid]/edit/+page.ts`
    - In `ui/src/routes/pos/[poid]/add-child/+page.ts`
  - Doesn't consider the current user's permission level
  - Doesn't account for auto-approval scenarios
  - Will be replaced by the new dynamic `/api/purchase_orders/approvers/{amount}` endpoint

## API Endpoints

We will implement two new endpoints:

### 1. GET /api/purchase_orders/approvers/{division}/{amount}

Returns a list of users who can serve as first approvers for a purchase order with the specified amount and division.

**Behavior:**

- If user has no approver claims: Return all users with po_approver claim
- If user has po_approver claim or higher: Return empty list (will auto-set to self in UI)
- Results are filtered to only include approvers who have permission for the specified division. Permissions are determined by a claim payload. No payload means full permission, divisions in the payload restrict permissions to those divisions.

**Response Format:**

```json
[
  { "id": "user123", "given_name": "John", "surname": "Doe" },
  { "id": "user456", "given_name": "Jane", "surname": "Smith" }
]
```

### 2. GET /api/purchase_orders/second_approvers/{division}/{amount}

Returns a list of users who can serve as second approvers for a purchase order with the specified amount and division.

**Behavior:**

- If amount < tier1: Return empty list (no second approval needed)
- If tier1 ≤ amount < tier2:
  - If user has po_approver_tier2 claim or higher: Return empty list (will auto-set to self in UI)
  - Otherwise: Return all users with po_approver_tier2 claim
- If amount ≥ tier2:
  - If user has po_approver_tier3 claim: Return empty list (will auto-set to self in UI)
  - Otherwise: Return all users with po_approver_tier3 claim
- The code should intelligently adjust to even greater numbers of tiers beyond tier3 if they exist in the database by simply sorting them by max_amount
- Results are filtered to only include approvers who have permission for the specified division, with the same behaviour as in the approvers endpoint.

**Response Format:**

```json
[
  { "id": "user123", "given_name": "John", "surname": "Doe" },
  { "id": "user456", "given_name": "Jane", "surname": "Smith" }
]
```

## Implementation Locations

### Backend Changes

1. **New Route Handlers**:
   - Add the two new route handlers in `app/routes/purchase_orders.go`

   ```go
   // GetApprovers returns a list of users who can approve a purchase order of the given amount and division
   func GetApprovers(c echo.Context) error {
       division := c.Param("division")
       amount := c.Param("amount")
       
       // Implementation details...
   }

   // GetSecondApprovers returns a list of users who can provide second approval for a purchase order of the given amount and division
   func GetSecondApprovers(c echo.Context) error {
       division := c.Param("division")
       amount := c.Param("amount")
       
       // Implementation details...
   }
   ```

   - Alternatively these could be implemented a single function GetApprovers with an argument specifying the behaviour mode outlined above.

2. **Route Registration**:
   - Register the new routes in `app/routes/routes.go`

   ```go
   // Add these routes inside RegisterPurchaseOrderRoutes
   g.GET("/purchase_orders/approvers/:division/:amount", purchase_orders.GetApprovers)
   g.GET("/purchase_orders/second_approvers/:division/:amount", purchase_orders.GetSecondApprovers)
   ```

3. **Purchase Order Validation**:
   - Update validation logic in `app/hooks/purchase_orders.go` to enforce rules about the priority_second_approver field

4. **Approval Process Update**:
   - Modify the approval logic in `app/routes/purchase_orders.go` to consider the priority_second_approver field and the 24-hour window

### Frontend Changes

1. **PurchaseOrdersEditor Component**:
   - Update `ui/src/lib/components/PurchaseOrdersEditor.svelte` to:
     - Fetch approver options based on PO amount. For recurring POs this will be the product of the number of recurrences and the total. Code exists for this in the back end (go app) but may not exist in the front end. We should verify if it exists or not and if it doesn't create it for the front end to mirror the behaviour of the back end.
     - Dynamically show/hide fields based on user permissions
     - Handle auto-selection logic

2. **Purchase Order Pages**:
   - Update the following files to use the new API endpoints instead of the po_approvers view:
     - `ui/src/routes/pos/add/+page.ts`
     - `ui/src/routes/pos/[poid]/edit/+page.ts`
     - `ui/src/routes/pos/[poid]/add-child/+page.ts`

3. **API Client Types**:
   - Update types in `ui/src/lib/pocketbase-types.ts` to remove references to the redundant views

## Migration Considerations

1. The `po_approvers` view will be redundant and can be removed via a migration

2. Create a migration to update the documentation of the `priority_second_approver` field in the purchase_orders collection. (This was written, but why? Think this part through)

## Business Logic Details

### Division-Specific Approver Logic

Some approvers have division restrictions in their claim payloads:

1. Approvers with an empty payload can approve POs for any division
2. Approvers with a non-empty payload can only approve POs for divisions listed in their payload

When the optional division parameter is provided to the API endpoints:

- Only return users whose po_approver claim payload either:
  - Is empty (indicating they can approve for any division), or
  - Contains the specified division ID
- This ensures that users will only see approvers who are authorized for the division of the PO they're creating/editing

### 24-Hour Window Implementation

When a purchase order is created or updated with a priority_second_approver:

1. Record the timestamp of the assignment. For the first version of the implementation we'll just use PocketBase's built in 'updated' field of the purchase_orders record. This has the effect of resetting the 24-hour countdown every time the purchase_orders record is updated, but that's fine.
2. For 24 hours, only show this PO to the priority_second_approver for second approval
3. After 24 hours, if not approved, make it visible to all users with the appropriate claim
4. This can probably be implemented by just having the scheduled emailer (not implemented) and the query for the UI, check the timestamp mentioned in 1 and compare it to the current time.

Here's a possible implementation that needs to be verified:

```sql
SELECT * FROM purchase_orders 
WHERE 
  status = 'Unapproved' AND
  LENGTH(approved) > 0 AND -- The record already has first-level approval
  (
    priority_second_approver = {:userId} OR -- caller is the priority second approver
    priority_second_approver = "" OR -- no specified priority second approver
    priority_second_approver IS NULL OR
    (
      -- There is a priority second approver and it's not the caller, but the 24
      -- window for exclusivity has closed
      priority_second_approver IS NOT NULL AND
      priority_second_approver != "" AND
      priority_second_approver != {:userId} AND
      updated < datetime('now', '-24 hours') -- SQLite Syntax
    )
  )
```

It's important to note that all status = 'Active' purchase orders can be viewed
by anybody who is authenticated. It's the ones with a different status that need
special filtering. So the above query will actually be implemented as pocketbase
listRule and viewRule strings.

```rules
// Active purchase_orders can be viewed by any authenticated user
(status = "Active" && @request.auth.id != "") ||

// Cancelled and Closed purchase_orders can be viewed by uid, approver, second_approver, and 'report' claim holder
(
  (status = "Cancelled" || status = "Closed") &&
  (
    @request.auth.id = uid || 
    @request.auth.id = approver || 
    @request.auth.id = second_approver || 
    @request.auth.user_claims_via_uid.cid.name ?= 'report'
  )
) ||

// TODO: We may also later allow Closed purchase_orders to be viewed by uid, approver, and committer of corresponding expenses, if any, in a rule here
(
  true = true
) ||

// Unapproved purchase_orders can be viewed by uid, approver, priority_second_approver and, if updated is more than 24 hours ago, any holder of second_approver_claim
(
  status = "Unapproved" &&
  (
    @request.auth.id = uid || 
    @request.auth.id = approver || 
    @request.auth.id = priority_second_approver || 
    (
      // updated more than 24 hours ago and @request.auth.id holds second_approver_claim
      updated < @yesterday && @request.auth.user_claims_via_uid.cid ?= second_approver_claim
    )
  )
)
```

#### The alternative PO implementation with no tiers

In the alternative PO implementation, there is no second_approver_claim field. Instead, we will need to find a different way to validate whether a user can view/list the PO before the 24 hour window (they're the priority_second_approver, so that doesn't change) or whether the user can view/list the PO after the 24 hour window has expired. In this second case, we can't check that the user has the second_approver_claim. Instead we need to test several things:

1. The user must have the po_approver claim
2. Their po_approver claim's payload.max_amount >= the amount of the purchase_order
3. Their po_approver claim's payload.max_amount <= the value of lowest po_approval_threshold value that is greater or equal to the approval_amount of the purchase_order
4. Their po_approver claim's payload.divisions is missing or includes the division id of the purchase_order

### Auto-Approval Logic

Existing auto-approval logic is set to FALSE by constant. Maintain the existing auto-approval logic, but ensure that it is updated to reflect the new code:

1. If the creator has sufficient claims, the PO is auto-approved
2. Add logic to use the priority_second_approver for auto-setting the second_approver when appropriate
