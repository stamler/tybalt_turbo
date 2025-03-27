# Purchase Order Priority Second Approver Feature

## Purpose

When a purchase order requires second approval based on its value, any user with a max_amount greater than the approval_total and no divisions restrictions may perform the second approval. This would result in all users with necessary permissions seeing many purchase orders in their approval queue, creating noise and potential confusion over responsibility.

The priority_second_approver field addresses this by:

1. Allowing the creator/editor to specifically designate a second approver
2. Creating a 24-hour window of exclusive review for the designated approver
3. Falling back to the standard all-users approach only if the designated approver doesn't act within the window

## API Endpoints

The UI loads approvers and second approvers from two endpoints.

### 1. GET /api/purchase_orders/approvers/{division}/{amount}

Returns a list of users who can serve as first approvers for a purchase order with the specified amount and division.

**Behaviour:**

- If user has no approver claims: Return all users with po_approver claim and permission based on the submitted division.
- If user has po_approver claim or higher and not restricted by division: Return empty list (will auto-set to self in UI)
- Results are filtered to only include approvers who have permission for the specified division. Permissions are determined by the `po_approver_props` table. An empty divisions array means full permission, divisions is a whitelist.

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

- If amount < lowest threshold in `po_approval_thresholds` table, return empty list (no second approval needed)
- Otherwise return users with max_amount >= approval_total but less than or equal to the next biggest threshold
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

Some approvers have division restrictions in their po_approval_props:

1. Approvers with an empty payload can approve POs for any division
2. Approvers with a non-empty payload can only approve POs for divisions listed in their payload

The division parameter provided to the API endpoints only return users where the divisions list:

- Is empty (indicating they can approve for any division), or
- Contains the specified division ID

Ensuring that callers will only see approvers who are authorized for the division of the PO they're creating/editing

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
