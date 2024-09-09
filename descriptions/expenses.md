# The expenses system

## Summary of the Expenses System

The expenses system depends on the purchase_orders system and the job categories system. So these must be implemented first.

The user can create, edit, delete, and submit expenses with the expenses system. Additionally, the user can view a list of all their expenses. Users who are managers can approve or reject expenses submitted by their direct reports. When an expense is approved, the approved property is set to the current timestamp, otherwise the expense record's timestamp value is an empty string. When an expense is rejected, the rejected flag is set to true and a rejection reason is provided. The id of the who rejected the expense is recorded in the rejector property.

## Pocketbase Collection Schema (expenses)

- date (string, required, YYYY-MM-DD)
- pay_period_ending (string, required, YYYY-MM-DD, saturday)
- uid (references users collection, required)
- approver (references users collection, required)
- approved (datetime)
- division (references divisions collection, required)
- job (references jobs collection)
- payment_type (enum, required) [Allowance, Expense, FuelCard, Mileage, CorporateCreditCard, PersonalReimbursement] **DEPRECATE FuelOnAccount (do not replace), Meals (replace with Allowance)**
- allowance_types (Multiple Enum OR JSON, required if payment_type is Allowance, else empty) [Lodging, Lunch, Dinner, Breakfast]
- total (number, required)
- submitted (datetime)
- committer (references users collection)
- committed (datetime)
- committed_week_ending (string, required, YYYY-MM-DD, saturday)
- description (string, minimum 5 characters)
- rejected (boolean)
- rejector (references users collection)
- rejection_reason (string, minimum 5 characters)
- distance (number)
- vendor_name (string)
- attachment (file)
- cc_last_4_digits (string)
- po (references purchase_orders collection)
- category (references the categories set of the corresponding job)

## The expense entry/edit page

## The expense list page
