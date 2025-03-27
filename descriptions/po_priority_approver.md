# Purchase Order Priority Second Approver Feature

## Overview

This document outlines the implementation of the priority second approver feature for purchase orders. This feature allows users to specify which holder of the appropriate approval tier claim should be responsible for providing the second approval on a purchase order.

## Purpose

When a purchase order requires second approval based on its value, the current system does not designate a specific user to handle that approval. This results in all users with required permissions seeing the purchase order in their approval queue, creating unnecessary noise and potential confusion over responsibility.

The priority_second_approver field addresses this by:

1. Allowing the creator/editor to specifically designate a second approver
2. Creating a 24-hour window of exclusive review for the designated approver
3. Falling back to the standard all-users approach only if the designated approver doesn't act within the window

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
3. After 24 hours, if not approved, make it visible to all users with the appropriate permissions
4. This can probably be implemented by just having the scheduled emailer (not implemented) and the query for the UI, check the timestamp mentioned in 1 and compare it to the current time.

It's important to note that all status = 'Active' purchase orders can be viewed
by anybody who is authenticated. It's the ones with a different status that need
special filtering. So the above query will actually be implemented as pocketbase
listRule and viewRule strings.

We need to find a way to validate whether the user can view/list the PO after the 24 hour window has expired. To accomplish this, we need to test several things:

1. The user must have the po_approver claim
2. @request.auth.user_claims_props_po_approver.max_amount >= the approval_total of the purchase_order. This ensures that the caller has the permission to actually approve/reject the po in question based on amount.
3. Their po_approver claim's payload.max_amount <= the value of lowest po_approval_threshold value that is greater or equal to the approval_amount of the purchase_order. We do this by joining a view collection `purchase_orders_augmented` which has the lower and upper thresholds included as columns for each po and testing whether the user's max_amount is less than the upper_threshold. In doing this we indirectly filter out POs that, even though the max_amount is greater than the approval_total, can be approved by a lower qualified user in a lower tier.
4. Their po_approver claim's payload.divisions is missing or includes the division id of the purchase_order

#### purchase_orders_augmented view

```sql
SELECT 
    po.id, po.approval_total,
    COALESCE((SELECT MAX(threshold) 
     FROM po_approval_thresholds 
     WHERE threshold < po.approval_total), 0) AS lower_threshold,
    COALESCE((SELECT MIN(threshold) 
     FROM po_approval_thresholds 
     WHERE threshold >= po.approval_total),1000000) AS upper_threshold,
    (SELECT COUNT(*) 
     FROM expenses 
     WHERE expenses.purchase_order = po.id AND expenses.committed != "") AS committed_expenses_count
FROM purchase_orders AS po;
```

#### additional listRule and viewRule fragments

```rules
...exiting rules
// Unapproved purchase_orders can be viewed by uid, approver, priority_second_approver and, if updated is more than 24 hours ago, any holder of the po_approver claim whose po_approver_props.max_amount >= approval_amount and <= the upper_threshold of the tier.
(
  status = "Unapproved" &&
  (
    @request.auth.id = uid || 
    @request.auth.id = approver || 
    @request.auth.id = priority_second_approver 
  ) || 
  (
    // updated more than 24 hours ago
    updated < @yesterday && 
    
    // caller has the po_approver claim
    @request.auth.user_claims_via_uid.cid.name ?= "po_approver" &&

    // caller max_amount for the po_approver claim >= approval_total
    @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount >= approval_total &&

    // caller user_claims.payload.divisions = null OR includes division
    (
      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:length = 0 ||
      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:each ?= division
    ) &&
    (
      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount >= approval_total &&
      @collection.purchase_orders_augmented.id ?= id &&
      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount ?<= @collection.purchase_orders_augmented.upper_threshold
    )
  )
)
```
