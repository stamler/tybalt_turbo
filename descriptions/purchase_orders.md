# The Purchase Orders System

## Purchase Order types and their behaviours

| PO Type    | Description                                                                                   | May be Closed Manually                                               | May be Canceled if status is Active                              | Closed automatically                    | Can be converted to different PO type                                         | Approval tier required                                |
|------------|-----------------------------------------------------------------------------------------------|----------------------------------------------------------------------|------------------------------------------------------------------|-----------------------------------------|-------------------------------------------------------------------------------|-------------------------------------------------------|
| Normal     | Valid for a single expense                                                                    | No                                                                   | Yes, if no expenses are associated                               | When an expense is committed against it | Yes, to Cumulative by payables_admin claim holder                             | Based on PO amount                                    |
| Recurring  | Valid for a fixed number of expenses not exceeding specified value                            | Yes, if status is Active and > 0 expenses associated                 | Yes, if no expenses are associated                               | When the final expense is committed     | No                                                                            | Based on PO amount * number of periods                |
| Cumulative | Valid for an unlimited number of expenses where sum of values does not exceed specified value | Yes, if status is Active and > 0 expenses associated                 | Yes, if no expenses are associated                               | When the maximum amount is reached      | No                                                                            | Based on PO amount                                    |
|            |                                                                                               | createClosePurchaseOrderHandler (app/routes/close_purchase_order.go) | createCancelPurchaseOrderHandler (app/routes/purchase_orders.go) | n/a (handled in expense processing)     | createConvertToCumulativePurchaseOrderHandler (app/routes/purchase_orders.go) | getSecondApproverClaim (app/hooks/purchase_orders.go) |

## Approvals

All purchase orders will have a status of `Unapproved` when they are created.
Approval thresholds are stored in the po_approval_thresholds table in the
database. The thresholds are used to group together purchase_orders record's
approval_total values into tiers. The lower threshold is always less than the
approval_total while the upper threshold is greater than or equal to the
approval_total. approval_totals greater than the lowest threshold in the
po_approval_thresholds table will always require a second approval. This is
determined in the GetPOApprovers() function (for assigning a priority second
approver) and in the createApprovePurchaseOrderHandler() function (for not
marking a record as 'Active' if the first user to approve it doesn't have a
max_amount that is creater than or equal to the approval_total). Note that the
one user can be both the approver and the second_approver if they pass the
requirements for both.

### Division-Specific Approver Logic

Some approvers have division restrictions in their po_approval_props:

1. Approvers with an empty payload can approve POs for any division
2. Approvers with a non-empty payload can only approve POs for divisions listed in their payload

The division parameter provided to the API endpoints only return users where the divisions list:

- Is empty (indicating they can approve for any division), or
- Contains the specified division ID

Ensuring that callers will only see approvers who are authorized for the division of the PO they're creating/editing

### priority_second_approver

When a purchase order requires second approval based on its value, any user with a max_amount greater than the approval_total and no divisions restrictions may perform the second approval. This would result in all users with necessary permissions seeing many purchase orders in their approval queue, creating noise and potential confusion over responsibility.

The priority_second_approver field addresses this by:

1. Allowing the creator/editor to specifically designate a second approver
2. Creating a 24-hour window of exclusive review for the designated approver
3. Falling back to the standard all-users approach only if the designated approver doesn't act within the window

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

## Description of the Purchase Orders System

Any user can create a purchase_order, and their uid is in the uid column of the
purchase order record. A purchase order can be of type `Normal`, `Cumulative`,
or `Recurring`. A `Normal` purchase order is valid for one expense and then is
no longer valid. A `Cumulative` purchase order is valid for a number of expenses
until the total amount of the expenses reaches the total of the purchase order.
Once that total is reached, the purchase order is automatically closed during
commit. A `Recurring` purchase order is valid for multiple expenses at a
specified frequency until a specified end date, and once that date is reached,
the purchase order is automatically closed during commit. A purchase order has a
`payment_type` that specifies how the associated expense(s) is/are to be paid. A
purchase order may have an attachment, which is a file. How this will be handled
in the by the API endpoint is an open problem that we will need to addressed. A
purchase order creator specifies the approver from a list which is determined by
the backend. Approval is handled by an api endpoint that is invoked by a user.
The payload of the request will contain the id of the purchase order and the
endpoint will determine whether to set the approved timestamp, second_approval
timestamp, or both based on the id of the user making the request and the
user_claims and `po_approver_props` stored in the database. If the
purchase_order is approved by the approver and, if applicable, the second
approver, a po_number is generated assigned by the system and written into the
po_number column of the purchase order record. The approver or second approver
(if applicable) can reject the purchase_order. If the purchase_order is
rejected, the rejecting user's id is recorded in the rejector property. The
rejection_reason is provided by the rejecting user. The rejected field is set to
the current date and time. The purchase order's status remains `Unapproved`.

If a purchase order's approval_total is greater than the lowest threshold in the
database, second approval is required. The second approver must hold a
max_amount that is greater than or equal to the purchase order's approval_total.
Additionally the second approver must either be allowed to approve purchase
orders in any division, or their divisions whitelist must contain the division
of the purchase order pending approval. A purchase order must be fully approved
prior having its status set to Active.

## Lifecycle of a Purchase Order

1. A purchase order is created by a user.
2. The creator may delete a purchase order as long as it is not `Active`.
3. The purchase order is either approved or rejected by the approver the user has specified
4. If the purchase order requires second approval, it is either approved or rejected by the second approver
5. The purchase order is `Active`

## Cancellation

The user with uid, approver or second approver (if applicable) can cancel the
purchase_order if it has no expenses against it. When a purchase_order is
cancelled, the cancelling user's id is recorded in the canceller property. The
cancelled property of the purchase order is set to the current date and time.
The purchase order's status is set to Cancelled.

## API Endpoints

These endpoints should be implemented in app/routes/purchase_orders.go and called from app/routes/routes.go

- POST /api/purchase_orders/:id/approve # approve po with :id, based on caller's id and claims

  This endpoint handles the approval of a purchase order. All database
  operations are performed within a single transaction. If the record doesn't
  have a status of Unapproved, the call fails. If the record's rejected property
  is not blank, the purchase order was already rejected and the call fails. The
  endpoint completes both first and second approvals in one call if the caller
  is so qualified. If the caller's id matches the approver property of the
  purchase order record, then this is the (first, primary) approver. In this
  case, the approved property is updated with the current timestamp, and the
  status is set to Active. Otherwise, we check if the caller is a qualified
  second approver by checking whether their max_amount is greater or equal to
  the approval_total of the purchase order and whether the caller has any
  divisional restrictions. If the caller is qualified, a po_number is generated
  in the format YYYY-NNNN where NNNN is a sequential zero-padded 4 digit number
  beginning at 0001 and less than 5000. We query the existing PO numbers in the
  purchase_orders collection to ensure the generated PO number is unique. If the
  caller is not a qualified second approver, the call fails with a permission
  denied error.

- POST /api/purchase_orders/:id/reject # reject po with :id based on caller's id, JSON body contains rejection reason

  This endpoint handles the rejection of a purchase order. All database
  operations are performed within a single transaction. If no rejection_reason
  property exists in the payload, the call fails with a bad request error. If
  the status of the purchase order is not Unapproved, the call fails with a bad
  request error. If the caller's id matches the approver of the purchase order,
  we update the rejection_reason property with the payload's rejection_reason
  and the rejector property with the caller's id. The rejected timestamp is set
  to the current date and time. The status remains Unapproved, and the api call
  returns with success. Otherwise, we must check if the caller is a qualified
  second approver. To do this we first query the user_claims collection for the
  caller's claims. The caller is a qualified second approver if their max_amount
  is greater or equal to the approval_total of the purchase order and they have
  no divisional restrictions.

- POST /api/purchase_orders/:id/cancel # cancel po with :id based on caller's id

  This endpoint handles the cancellation of a purchase order. All database
  operations are performed within a single transaction. If the status of the
  purchase order is not `Active`, the call fails. The purchase_order may not
  have any associated expenses. If the caller's id matches the uid in the
  purchase order record, the canceller is set to the caller's id and the
  cancelled property is set to the current timestamp. The status is set to
  Cancelled. The endpoint returns a success code if the purchase order is
  cancelled, otherwise it returns an error.

- POST /api/purchase_orders/:id/close # close po with :id based on caller's id

  This endpoint handles the closure of a purchase order. Only `Recurring` and
  `Cumulative` purchase orders may be manually closed. A purchase order must
  have at least one associated expense to be closed, otherwise it may be
  cancelled instead.

## Pocketbase Collection Schema (purchase_orders)

This schema should be implemented in a go migrations file in app/migrations/

- po_number (string in the format YYYY-NNNN where NNNN is a sequential
  zero-padded 4 digit number beginning at 0001 and less than 5000, unique,
  required if status is Active or Cancelled, must be blank if status is
  Unapproved)
- status (enum) [Unapproved, Active, Cancelled]
- uid (references users collection, required)
- type (enum, required) [normal, cumulative, recurring]
- date (string, required, YYYY-MM-DD)
- end_date (string, required if type is recurring, YYYY-MM-DD)
- frequency (enum, required if type is recurring) [Weekly, Biweekly, Monthly]
- division (references divisions collection, required)
- description (string, minimum 5 characters)
- total (number, required)
- payment_type (enum, required) [OnAccount, Expense, CorporateCreditCard]
- vendor_name (string, required)
- attachment (file)
- rejector (references users collection)
- rejected (datetime)
- rejection_reason (string, minimum 5 characters)
- approver (references users collection, required)
- approved (datetime)
- second_approver (references users collection)
- second_approval (datetime, interpret as date the purchase order is created)
- canceller (references users collection)
- cancelled (datetime)
