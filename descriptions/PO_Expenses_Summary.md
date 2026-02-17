# PO / Expenses System Summary (Current)

## 1. Anyone can create a `purchase_orders` record

PO creation is open to authenticated users; approval gates activation.

## 2. Purchase order approval is stage-based (not threshold-tier table based)

1. Dual approval is required only when:
   - `expenditure_kinds.second_approval_threshold > 0`, and
   - `approval_total > second_approval_threshold`.

2. `approval_total` is:
   - `total` for `One-Time` and `Cumulative`
   - computed full-period value for `Recurring`

3. Dual-required POs must have:
   - a valid first-stage `approver`
   - a valid `priority_second_approver`

4. Approval route behavior:
   - Stage 1 sets `approved` and keeps status `Unapproved` for dual-required POs
   - Stage 2 sets `second_approval` and activates PO
   - Bypass fast path may set both stages in one call when caller is second-stage qualified

5. PO becomes `Active` and gets `po_number` only when all required approval stages are complete.

## 3. Expenses and purchase orders coupling

An expense may require a PO based on existing payment-type and threshold rules. If a PO is provided, it must be `Active`.

When an expense references a PO:

1. Expense date constraints are validated against PO date/end_date
2. Type-specific total/overflow validations are enforced
3. Expense kind is derived from PO kind

## 4. Authorization note for expense submission

Current policy allows users to submit expenses against any `Active` PO, subject to validation rules above.
