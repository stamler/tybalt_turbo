# Expenditures

| Activity                               | All Staff   | Managers / Team Leads | Branch Managers | Executives |
| -------------------------------------- | ----------- | --------------------- | --------------- | ---------- |
| Capital Expense                        | < 100$      | < 500$ w/PO           | < 2500$ w/PO    | w/PO       |
| Project Expense (references a job)     | < 100$ w/PO | < 500$ w/PO           | < 20000$ w/PO   | w/PO       |
| Sponsorships                           |             |                       |                 | w/PO       |
| Staff Appreciation or Social Committee |             |                       | w/PO            | w/PO       |
| Media Advertising and Event Fees       |             |                       |                 | w/PO       |
| Computer and Software Acquisition      |             |                       |                 | w/PO       |

## Implementation Notes

Each of the 6 activities maps to columns in `po_approver_props`. `standard` (Capital Expense) uses `max_amount` with no job and `project_max` when a job is present. Other kinds use dedicated columns: `sponsorship_max`, `staff_and_social_max`, `media_and_event_max`, `computer_max`.

In the UI:

- **Purchase Orders editor**: Kind is always selectable via toggle. If selected kind has `allow_job = false`, Job is hidden and cleared; switching back to an `allow_job = true` kind restores the in-editor Job value.
- **Expenses editor**: Kind is never selectable. It is display-only: inherited from the linked PO when a PO is present, otherwise shown as `standard` for no-PO expenses.

Operational dependency:

- Approval eligibility depends on synchronized `po_approver_props` data. In mixed Turbo/legacy operation, authoritative values flow through `poApproverProps` writeback -> Firestore `TurboPoApproverProps` -> MySQL `TurboPoApproverProps` -> `import_data --users` (Turbo values take precedence per user, synthesis is fallback only).
