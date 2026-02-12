# Alternative purchase_orders record approval system

1. The value of a `purchase_orders` record is the `total` field. This amount
   cannot be used for `Recurring` purchase orders as their value for approval is
   determined as the product of the total and the number of recurring payments.
   Another field, `approval_total`, populated by the create/update hook for
   `purchase_orders`, simply copies the value for `One-Time` and `Cumulative`
   purchase orders. But for `Recurring` purchase orders, this `approval_total`
   is calculated as the aforementioned product.
2. All `purchase_orders` approval permissions are determined by a single
   `po_approver` claim and associated payload
3. The payload of the `po_approver` claim contains a JSON object with 2
   properties
   - `max_amount`: an number above which the user is not permitted to _fully_
     approve a purchase order
   - `divisions`: an optional list of strings representing `divisions` ids the
     user is permitted to approve purchase_orders records for. If empty, the
     user can approve `purchase_orders` records whose `approval_total` is less
     than or equal to their `max_amount` for all divisions.
4. The `po_approval_tiers` table is deleted
5. The `po_approval_thresholds` table is created. It just has two columns,
   `threshold`, and `description`, plus a unique `id`. Normally it would have
   just one row, where the threshold value is the amount above which second
   approval is required. If it has more than one row, the lowest value is used
   in determining this threshold for second approval requirements and subsequent
   higher thresholds are used only to restrict visibility of purchase_orders
   requiring second approval in order to reduce noise for users with higher
   levels of approval authority. Concretely, users with a max_amount in their
   claim payload will only be shown unapproved purchase_orders records whose
   approval_total falls between the same two thresholds as their po_approver
   claim's max_amount payload value.
6. All `purchase_orders` records with values above the the lowest `max_amount`
   of the `purchase_orders_tiers` table require two approvers
7. The first approver of a `purchase_orders` record is selected by the creator
   from a list of all users with the `po_approver` claim. Permitted approvers
   must either have no value for their `divisions` payload, or the list of
   `divisions` must contain the id of the `division` of the `purchase_orders`
   record being approved.
8. If required, the `priority_second_approver` is selected from users whose
   `po_approver` payload `max_amount` is greater than or equal to the amount of
   the `purchase_orders` record. The `priority_second_approver` has a 24 hour
   timeout of exclusivity for approval, after which, all `po_approver` holders
   with `max_amounts` greater than or equal to the `purchase_orders` record
   amount will be able to view and approve the `purchase_orders` record.
   Division restrictions apply.
9. Fully approved (meaning second approval is present if required)
   `purchase_orders` cannot be deleted, only `Closed` or `Cancelled`.
