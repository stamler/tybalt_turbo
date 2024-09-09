# The Purchase Orders System

## Constants

MANAGER_PO_LIMIT = 500
VP_PO_LIMIT = 2500

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
or its total is greater than or equal to the VP_PO_LIMIT, the second approver
must be a member of the SMG group and the second_approver_claim will be set
accordingly. Otherwise if the purchase order's total is greater than or equal to
the MANAGER_PO_LIMIT, the second approver must be a member of the VP group and
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

## Pocketbase Collection Schemas (purchase_orders)

- po_number (string, unique, required if status is Active or Cancelled, must be blank if status is Unapproved)
- status (enum) [Unapproved, Active, Cancelled]
- type (enum, required) [normal, cumulative, recurring]
- date (string, required, YYYY-MM-DD)
- end_date (string, required if type is recurring, YYYY-MM-DD)
- frequency (enum, required if type is recurring) [Weekly, Biweekly, Monthly]
- uid (references users collection, required)
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
