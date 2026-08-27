# Purchase Order Approver Lists

This document explains how Turbo populates the Primary Approver list and the Second Approver list.

## Source functions

`GetPOApproverPolicy` in `app/utilities/po_approvers.go` builds the two approver pools:

- `FirstStageApprovers` supplies the Primary Approver list.
- `SecondStageApprovers` supplies the Second Approver list.

`createGetApproversHandler` in `app/routes/purchase_orders.go` prepares the purchase order data. It then calls `GetPOApproverPolicy` and returns the applicable list.

## Prepare the purchase order amount

Before Turbo builds the lists, Turbo does these actions:

1. Turbo validates the division, amount, expenditure kind, and job status.
2. For a recurring purchase order, Turbo calculates the total scheduled value.
3. Turbo converts a foreign-currency amount to the home-currency amount.
4. Turbo selects the approval-limit field for the expenditure kind.

| Expenditure kind | Approval-limit field |
| --- | --- |
| Capital | `max_amount` |
| Project | `project_max` |
| Sponsorship | `sponsorship_max` |
| Staff and social | `staff_and_social_max` |
| Media and event | `media_and_event_max` |
| Computer | `computer_max` |

## Find eligible approvers

Turbo first finds all users who meet these conditions:

- The user has the `po_approver` claim.
- The user is active.
- The user has a positive limit for the applicable expenditure kind.
- The user is authorized for the selected division. An empty division list gives authorization for all divisions.

Turbo sorts these users by surname and then by given name.

## Determine the number of approval stages

Turbo requires second approval only when both conditions are true:

- The expenditure kind has a second-approval threshold greater than zero.
- The purchase order amount is greater than that threshold.

## Populate the Primary Approver list

If second approval is not required, the primary approver gives final approval. Turbo includes only users whose limit is equal to or greater than the purchase order amount.

If second approval is required, Turbo includes users whose positive limit is less than the purchase order amount. These users perform the first-stage review. Their limit can be greater than the second-approval threshold.

This design lets staff who are closer to the purchase vet the purchase order before it goes to the second approver. These staff can check the need for the purchase and the details of the request. The second approver can be further removed from the purchase and gives final financial approval.

This design also reduces approval noise. Approval noise means too many unnecessary requests for users who have higher approval limits. The first-stage review helps these users focus on requests that need their financial authority.

Turbo has one requester exception. For a purchase order that requires second approval, Turbo also includes the requester when the requester can give final approval. This exception lets the requester assign the purchase order to themselves.

## Populate the Second Approver list

Turbo populates this list only when second approval is required. Turbo includes users whose limit is equal to or greater than the purchase order amount.

These users can give final financial approval.

If the requester is a valid second approver, the endpoint returns an empty list. This result tells the user interface that the requester qualifies. For the requester's own purchase order, the user interface assigns the requester automatically.

If second approval is required and no valid second approver exists, Turbo returns the `second_pool_empty` error.

## Important result

The two pools are separate. A user with a positive limit below the purchase order amount can appear only in the Primary Approver list. A user whose limit covers the purchase order amount can appear only in the Second Approver list. The requester exception can also add a final-qualified requester to the Primary Approver list.
