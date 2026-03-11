# PO and Expense FAQ

This document is the FAQ for the current Turbo PO and expense workflow.

## Approval limits

**How do we know approval limits?**

Turbo determines valid PO approvers from the configured approval limits and division access for each approver. Staff creating a PO only see approvers who are valid for that PO.

Current authority matrix:

- Capital: all staff under $100; designated personnel, branch administrators, managers, and team leads under $500 with a PO; branch managers under $2,500 with a PO; executives with a PO.
- Project: all staff under $100 with a PO; designated personnel, branch administrators, managers, and team leads under $5,000 with a PO; branch managers under $25,000 with a PO; executives with a PO.
- Sponsorship: branch managers within budget with a PO; executives with a PO. Turbo does not enforce the total sponsorship budget.
- Staff/social: branch managers under $500 with a PO; executives with a PO.
- Media/event: executives with a PO.
- Computer/software: branch managers and executives with a PO. Computer/software purchases can also require IT vetting.

## Approvers and approval flow

**Does submitting a PO also request approval, or does approval need to happen in advance?**

Saving a PO starts Turbo's approval workflow. It stays `Unapproved` until the required approver(s) approve it.

**Will Turbo automatically choose the approver?**

For POs, not by manager. Turbo either:

- auto-assigns you if you qualify for that approval stage, or
- shows the valid approver list for you to choose from.

For expenses, yes: the approver is automatically set to the submitter's manager.

**If an approver name appears automatically, should staff leave it?**

If Turbo auto-assigns an approver, that assignment is intentional. If it shows a selector, the options shown are valid approvers for that PO.

**Will staff be notified about PO approvals?**

- When a PO is created or re-submitted, the first approver is notified.
- If second approval is required, the priority second approver is notified after first approval.
- When the PO becomes `Active`, the creator is notified if someone else approved it.
- Approvers can also review items in the pending approval queue.

**When does a PO number get assigned?**

Only after all required approvals are complete and the PO becomes `Active`.

## Credit cards and using someone else's PO

**If I use someone else's company card, do I just enter the last 4 digits? Does that person have to create the PO?**

Enter the last 4 digits on the expense. The cardholder does not have to create the PO: any authenticated user can create a PO, and staff can submit an expense against any `Active` PO, including someone else's.

## Recurring POs

**What if a recurring PO does not have a known end date?**

Turbo does not support open-ended recurring POs. A recurring PO must have:

- an end date
- a frequency
- an end date after the start date
- an end date within 400 days of the start date
- at least 2 occurrences

## Vendors

**If the vendor search shows multiple similar results, does it matter which one is chosen?**

Choose an active vendor that matches the supplier. The code does not apply different approval behavior based on duplicate names.

## Total / max amount

**What does "max amount" mean for the total?**

- One-Time and Cumulative POs: the total is the maximum amount available on that PO.
- Recurring POs: the total is the amount per occurrence. Turbo uses the full scheduled value of the recurring PO for approval.
