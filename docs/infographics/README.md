# Turbo staff guide — draft 01

Prepared on 28 September 2026 from repository revision `306ef2f0`.

The guide explains how POs, expenses and time sheets relate. It is a draft for content and layout review. It does not set company policy or confirm live configuration.

## Files

- `turbo-staff-guide.drawio`: editable diagrams.net XML. Open it in diagrams.net. Text, boxes and arrows are separate editable objects.
- `turbo-staff-guide.svg`: editable vector master. Open it in Inkscape or another SVG editor.
- `turbo-staff-guide.png`: image preview, 1600 × 2260 pixels.

The SVG and diagrams.net files contain the same draft. Choose one as the master for later edits; they do not update each other automatically. The PNG is an export of the SVG.

## Content decisions

- Show PO approval, expense approval and time-sheet approval as separate paths.
- Connect the Active PO to the creation of its expense. A new expense made outside a PO does not provide a PO selector.
- Show the second PO approval step as conditional. The application can combine steps when its policy permits this. Do not imply that two different people are always required.
- Use the PO Approvers table for current limits by person, spending category and division. Do not reproduce the older role-based authority matrix from the PO description as current configuration.
- Show $100 CAD only as a configurable software default. The $99.99/$100 comparison assumes no job and a non-exempt payment type. The $500 × 12 example assumes a CAD recurring PO. Neither example is a live setting.
- The no-PO expense limit, category threshold for second approval, and each person's final approval limit are three different values.
- A positive category threshold triggers second approval only when the approval total exceeds it. A zero threshold means no second approval for that category.
- Foreign-currency PO approval uses the CAD approval total. Expense validation uses the CAD settled amount where available, or the applicable conversion. Foreign-currency expense settlement must be complete before commitment.
- Use the application's payment-type names. `Expense` is the out-of-pocket path. `Personal Reimbursement` is a separate option that must be enabled for the user.
- A job is the common project reference. A corporate card is a payment type and does not remove PO requirements.
- Commitment finalises records for later processing. It does not prove that a payment has been made.

## Sources checked

Paths are relative to the repository root.

| Topic | Source |
| --- | --- |
| PO requirements, types and approval model | `descriptions/purchase_orders.md` |
| Expense creation, approval and correction | `descriptions/expenses.md` |
| Actual PO requirement checks, CAD amount comparison and document requirements | `app/hooks/validate_expenses.go` |
| Expense payment-type processing and Personal Reimbursement access | `app/hooks/expenses.go`, `ui/src/lib/expensePaymentTypes.ts` |
| Recurring approval total and currency conversion | `app/hooks/purchase_orders.go` |
| Approver eligibility and category thresholds | `app/utilities/po_approvers.go` |
| PO Approvers table, threshold help and access | `ui/src/lib/components/POApprovalLimits.svelte`, `app/routes/po_approval_limits.go` |
| Weekly time submission, assigned manager and project-authorisation check | `app/routes/bundle_timesheet.go` |
| Time entry fields and project context | `descriptions/time_entries.md` |
| Time-sheet actions and correction | `descriptions/time_sheets.md` |
| Approval and commitment | `app/routes/approve_record.go`, `app/routes/commit_record.go` |

## Review before staff publication

Confirm the live no-PO limit if the final guide is to state a fixed amount. Confirm the intended audience and who will help staff without access to PO Approvers. Decide whether this overview needs a separate guide for PO balances, overages, closure, rejected POs and later amendments; those details are outside this draft.

The draft files do not change application code, configuration or data.
