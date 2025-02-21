# The Purchase Orders System

## Constants

TIER_1_PO_LIMIT = 500
TIER_2_PO_LIMIT = 2500

## Description of the Purchase Orders System

Any user can create a purchase_order, and their uid is in the uid column of the
purchase order record. A purchase order can be of type normal, cumulative, or
recurring. A normal purchase order is valid for one expense and then is no
longer valid. A cumulative purchase order is valid for a number of expenses
until the total amount of the expenses reaches the total of the purchase order.
Once that total is reached, the purchase order is no longer valid. A recurring
purchase order is valid for multiple expenses at a specified frequency until a
specified end date, and once that date is reached, the purchase order is no
longer valid. A purchase order has a payment_type that specifies how the
associated expense(s) is/are to be paid. A purchase order may have an
attachment, which is a file. How this will be handled in the by the API endpoint
is an open problem that we will need to address. The purchase order record is
not directly created by the user, but rather is created by an API endpoint that
receives the purchase order data from the user. Upon creation the approver field
is set to the user's manager. Approval is handled by an api endpoint that is
invoked by a user. The payload of the request will contain the id of the
purchase order and the endpoint will determine whether to set the approved
timestamp, second_approval timestamp, or both based on the id of the user making
the request and the user_claims stored in the database. If the purchase_order is
approved by the approver and, if applicable, the second approver, a po_number is
generated assigned by the system and written into the po_number column of the
purchase order record. The approver or second approver (if applicable) can
reject the purchase_order. If the purchase_order is rejected, the rejecting
user's id is recorded in the rejector property. The rejection_reason is provided
by the rejecting user. The rejected is set to the current date and time. The
purchase order's status remains unapproved. The user with uid, approver or
second approver (if applicable) can cancel the purchase_order. Some purchase
orders require the approval of a second user. If the purchase order is recurring
or its total is greater than or equal to the TIER_2_PO_LIMIT, the second approver
must be a member of the SMG group and the second_approver_claim will be set
accordingly. Otherwise if the purchase order's total is greater than or equal to
the TIER_1_PO_LIMIT, the second approver must be a member of the VP group and
the second_approver_claim will be set accordingly. If neither of these
conditions are met, there is no need for a second approver so it will be left
blank. If the purchase_order is cancelled, the cancelling user's id is recorded
in the canceller property. The cancelled property of the purchase order is set
to the current date and time. The purchase order's status is set to Cancelled.
Depending on their contents, some purchase_orders will require additional
approval by users with other claims. These purchase orders will be submitted to
the next approver claim in the purchase_order record. A purchase order must be
fully approved prior having its status set to Active. Initially a purchase
order's status is set to Unapproved.

## Lifecycle of a Purchase Order

1. A purchase order is created by a user.
2. The purchase order is submitted for approval to the user's manager.
3. The purchase order is approved by the manager.
4. If conditions are met, the purchase order is submitted for approval to a VP
   or SMG member.
5. The purchase order is approved by the VP or SMG member (as applicable) and
   status set to Active.

At step 2 the purchase order can be deleted by the user.

At step 3 or 4 the purchase order can be rejected by the approver or second
approver (as applicable).

## API Endpoints

These endpoints should be implemented in app/routes/purchase_orders.go and called from app/routes/routes.go

- POST /api/po # create a new purchase order, JSON body + optional attachment

  This endpoint handles the creation of a new purchase order. All database
  operations are performed within a single transaction. The JSON body should
  include the necessary details including type, date, end date, frequency,
  division, description, total, payment type, vendor name, and attachment. The
  uid is populated from the context of the caller. The approver is set to the
  caller's manager from the profiles collection. The attachment is optional and
  can be processed as a file upload. The status is set to Unapproved. There
  should be no PO number at this stage and the purchase order should be added to
  the purchase_orders collection with a blank value for the po_number field. If
  the purchase order is recurring or if the total is greater than or equal to
  TIER_2_PO_LIMIT, the second approver claim is set to the id of the record in the
  claims collection with the name 'smg'. Otherwise, if the total is greater than
  or equal to TIER_1_PO_LIMIT, the second approver claim is set to the id of
  the record in the claims collection with the name 'vp'. If neither condition
  is met, the second approver claim is left blank. The endpoint returns the
  created purchase order record upon successful creation.

- PUT /api/po/:id # update the purchase order with :id, JSON body + optional new attachment file

  This endpoint handles the update of an existing purchase order. All database
  operations are performed within a single transaction. If the status is not
  Unapproved, the call fails. The caller's id must match the uid in the
  purchase order record. The JSON body should include the necessary details
  including type, date, end date, frequency, division, description, total,
  payment type, vendor name, and attachment. The attachment is optional and can
  be processed as a file upload. The status is set to Unapproved. There is no PO
  number at this stage. If the purchase order is recurring or if the total is
  greater than or equal to TIER_2_PO_LIMIT, the second approver claim is set to the
  id of the record in the claims collection with the name 'smg'. Otherwise, if
  the total is greater than or equal to TIER_1_PO_LIMIT, the second approver
  claim is set to the id of the record in the claims collection with the name
  'vp'. Otherwise, the second approver claim is left blank.

- DELETE /api/po/:id # delete the purchase order with :id if it's unapproved, caller's id must be the creator's uid

  This endpoint handles the deletion of an unapproved purchase order. All
  database operations are performed within a single transaction. Only purchase
  orders with a status of Unapproved can be deleted. The caller's id must be the
  uid in the purchase order record. The record is simply removed from the
  purchase_orders collection and the transaction is committed. The endpoint
  returns a success message if the purchase order is deleted, otherwise it
  returns an error.

- PATCH /api/po/:id/approve # approve po with :id, based on caller's id and claims

  This endpoint handles the approval of a purchase order. All database
  operations are performed within a single transaction. If the record doesn't
  have a status of Unapproved, the call fails. If the record's rejected property
  is not blank, the purchase order was already rejected and the call fails. The
  approver claim is used to determine whether this is the approver or second
  approver. If the caller's id matches the approver property of the purchase
  order record, then this is the (first, primary) approver. In this case, the
  approved property is updated with the current timestamp, and the status is set
  to Active. Otherwise, we check if the caller is a qualified second approver by
  loading the caller's claims from the user_claims collection. The caller is a
  qualified second approver if the purchase order's second_approver_claim
  property references one of the claims returned by the query to the user_claims
  collection. If so, we update the second_approval property with the current
  timestamp, and the status is set to Active. If the caller is not a qualified
  second approver, the call fails with a permission denied error.

- PATCH /api/po/:id/reject # reject po with :id based on caller's id, JSON body contains rejection reason

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
  caller's claims. The caller is a qualified second approver if the purchase
  order's second_approver_claim property references one of the claims returned
  by the query to the user_claims collection for the caller's claims. If this is
  so, the second_approver is set to the caller's id, the second_approval
  property is set to the current timestamp, and the status is set to Active.
  Finally a po_number is generated in the format YYYY-NNNN where NNNN is a
  sequential zero-padded 4 digit number beginning at 0001 and less than 5000. We
  query the existing PO numbers in the purchase_orders collection to ensure the
  generated PO number is unique.

- PATCH /api/po/:id/cancel # cancel po with :id based on caller's id

  This endpoint handles the cancellation of a purchase order. All database
  operations are performed within a single transaction. If the status of the
  purchase order is Unapproved, the call fails. If the caller's id matches the
  uid in the purchase order record, the canceller is set to the caller's id
  and the cancelled property is set to the current timestamp. The status is set
  to Cancelled. The endpoint returns a success code if the purchase order is
  cancelled, otherwise it returns an error.

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
- second_approver_claim (references claims collection)
- second_approver (references users collection)
- second_approval (datetime, interpret as date the purchase order is created)
- canceller (references users collection)
- cancelled (datetime)
