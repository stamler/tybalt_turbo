# WIP reports

Business contains one WIP button. It opens My WIP, Branch, and Division in a menu
to the right, or within the screen on mobile. The current view is highlighted.
The menu supports click, touch, Tab, arrow keys, and Escape. These reports include
Active jobs whose numbers do not start with P and whose project value is greater
than zero. Each row uses that job's own value and amounts. Parent and child jobs
remain separate. The report date is today in UTC, as on Job Details.

## Access and selection

- My WIP is always visible. It selects jobs whose manager is the caller. The API
  uses the authenticated user ID and accepts no manager override. An empty report
  explains that the caller has no qualifying projects.
- Branch WIP and Division WIP require the caller to manage any branch, or hold
  the kpi or admin claim. This grants access to all branches and divisions.
  The server checks access on every request. The links remain hidden from other
  users even when Show All UI is enabled.
- Branch WIP selects a branch the caller manages when available; otherwise it
  starts with All branches. All branches groups rows by branch, with a separate
  ranking in each group. A branch selection uses `jobs.branch`.
- Division WIP starts with the caller's default division. If none is set, it asks
  for a selection. The existing text autocomplete searches division code and name.
  A job must reference the division in `job_time_allocations`. By default, it must
  also have a time entry for that division with positive work hours through today.
  Committed and uncommitted time qualify; meal-only entries and amendments do not.
  The checkbox can include allocated jobs with no recorded work hours.
- Division selection only determines which jobs appear. Each row retains the
  whole job's WIP across all divisions.

## Values and ranking

All reports use the [Job WIP rules](job_wip.md). Time is always included. Include
committed expenses and Include active POs start checked. The PO expense deduction
always applies, even when expenses are excluded.

Rows with complete included values sort by percentage of project value, highest
first by default. The Sort by control also offers Remaining, lowest first.
Remaining is project value minus time and the selected expenses and PO balances.
Negative balances are red and sort before zero and positive balances. Sorting
uses signed dollar amounts, not their magnitudes. Equal values sort by job number,
then ID. Both sorts apply within each branch group. Percentages may exceed 100.
Percentage cells show the number over a bar. Bar width is limited to 0–100%,
but the number retains the actual value. Colour changes from green through
amber to red at 100%; values above 100% use purple. Incomplete values have no bar.
Estimated values remain ranked. Rows with unpriced included amounts appear in
Cannot rank with known values and no overall percentage or remaining balance. Excluding an incomplete
expense or PO factor restores ranking when the remaining factors are complete.
The report also counts otherwise matching jobs excluded for nonpositive project
values.

The compact table shows project number, client, project value, included WIP
factors, percentage, and remaining balance. The help popup explains how selected factors form the
percentage. Job numbers link to Job Details. A button beside each number opens the description, full client name, manager, and
branch. The document icon opens project details. In the Time column, `NR`
identifies a job with no rate sheet. `≈` identifies employee-rate estimates when
an assigned sheet cannot price some hours. `NR` replaces `≈` for jobs without a
sheet. A warning triangle still identifies incomplete values. Time indicators
appear before the amount, with the value indicator before `NR` when applicable.
Fully sheet-priced time needs no indicator.
Time popups show nonzero counts for missing entry roles, roles with no matching
sheet rate, employee-rate estimates, and unpriced hours. Excluded expense and PO columns are
hidden in these reports; Job Details retains its existing factor display.

The toolbar and inline title follow the PO Approvers layout. Search filters
client-side across project number, client, description, manager, and branch.
Every search word must match, without regard to case. Search includes rows in
Cannot rank and does not change their values or percentages.

## Shared code

`GET /api/wip/my`, `/api/wip/branch`, and `/api/wip/division` return WIP factors
for selected jobs with a positive project value in one pricing query. A small
scope query counts included and excluded jobs first; an empty included scope
skips pricing. Branch accepts an optional `branch` ID;
division requires `division` and accepts `require_time=true|false` (default true).
Responses contain `items`, `as_of`, and `excluded_project_value`.

The job and report queries provide `selected_jobs`, then share
`job_priced_time_entries.sql` and `job_wip.sql`. Staff and Divisions summaries use
the same pricing query. The shared `wipView` function supplies totals, completeness,
and percentages. The reports and Job Details share checkbox, help, missing-sheet,
and factor-notice components. Report controls rerank locally without changing
the server's PO balances.

## Validation

`app/wip_reports_test.go` uses append-only CSV fixtures for access, caller scope,
branch selection, division membership, recorded-hours filters, whole-job values,
parent/child separation, missing rates, unpriced amounts, and invalid requests.
The new job fixtures include authorization data to keep them out of unrelated
missing-authorization queues. Migration `1790712255_wip_lookup_indexes.go` adds
job/date expense lookups and positive-hour job/division/date time lookups through
the PocketBase collection schema. Its rollback removes only these two indexes.
See [WIP performance](wip_performance.md) for measurements and the repeatable
read-only benchmark.

UI tests are `npm run test:wip-reports` and `npm run test:wip-reports-browser`.
Browser tests cover ranking, notices, keyboard controls, navigation access,
selection defaults, filter changes, empty states, retry, stale responses, and
mobile layout. Existing WIP and time-summary tests cover the shared code.

## Missing rate details

The time popup has a `missing rates` button when recorded roles have no matching
sheet rate. Clicking it loads `GET /api/jobs/{id}/wip/missing-rates?as_of=YYYY-MM-DD`.
The authenticated endpoint uses the shared pricing query and the report date.
It returns the exact assigned sheet ID, name, and revision, plus affected roles
and their hours. Blank roles and future entries are excluded. Employee-rate
estimates and unpriced entries both contribute to the affected-role hours.

The popup expands in place and links the sheet name to its details page. It
shows loading, retry, and empty states, and keeps loaded results while the same
report data remains displayed. A changed report discards those results. The
query uses current job and sheet data; the report date limits entry dates, not
historical sheet assignments. Opening the popup alone makes no extra request.
