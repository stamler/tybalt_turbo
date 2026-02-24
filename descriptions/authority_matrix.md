# Expenditures

| Activity                               | All Staff   | Managers / Team Leads | Branch Managers                  | Executives |
|----------------------------------------|-------------|-----------------------|----------------------------------|------------|
| Capital Expense                        | < 100$      | < 500$ w/PO           | < 2500$ w/PO                     | w/PO       |
| Project Expense (references a job)     | < 100$ w/PO | < 5000$ w/PO          | < 25000$ w/PO                    | w/PO       |
| Sponsorships                           |             |                       | < budget<sup>1</sup> w/PO        | w/PO       |
| Staff Appreciation or Social Committee |             |                       | < 500$ w/PO                      | w/PO       |
| Media Advertising and Event Fees       |             |                       |                                  | w/PO       |
| Computer and Software Acquisition      |             |                       | w/PO and IT approval<sup>2</sup> | w/PO       |

- <sup>1.</sup> Turbo will enforce that an individual PO does not exceed the budget maximum, however turbo wil not enforce the budget so a branch manager could exceed the budget by approving multiple POs in a given year. It is the responsibility of the branch manager to manage this budget.
- <sup>2.</sup> It will be given an approval amount for computers up to $500 and thus can act as first approver and vet computer/software purchases for branch managers.

## Implementation Notes

Each of the 6 activities maps to columns in `po_approver_props`. `capital` uses `max_amount` (no job allowed). `project` uses `project_max` (job required). Other kinds use dedicated columns: `sponsorship_max`, `staff_and_social_max`, `media_and_event_max`, `computer_max`.

In the UI:

- **Purchase Orders editor**: Kind is always selectable via toggle. If selected kind has `allow_job = false`, Job is hidden and cleared; switching back to an `allow_job = true` kind restores the in-editor Job value.
- **Expenses editor**: Kind is never selectable. It is display-only: inherited from the linked PO when a PO is present, otherwise shown as `capital` for no-PO expenses.

Operational dependency:

- Approval eligibility depends on synchronized `po_approver_props` data. In mixed Turbo/legacy operation, authoritative values flow through `poApproverProps` writeback -> Firestore `TurboPoApproverProps` -> MySQL `TurboPoApproverProps` -> `import_data --users` (Turbo values take precedence per user, synthesis is fallback only).
