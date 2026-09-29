# WIP reports

Business contains My WIP, Branch WIP, and Division WIP. These reports include
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
first. Percentages may exceed 100. Equal percentages sort by job number, then ID.
Estimated values remain ranked. Rows with unpriced included amounts appear in
Cannot rank with known values and no overall percentage. Excluding an incomplete
expense or PO factor restores ranking when the remaining factors are complete.
The report also counts otherwise matching jobs excluded for nonpositive project
values.

The table shows project identity, client, manager, branch, project value, each WIP
factor, included value, and percentage. Job numbers link to Job Details. Shared
help controls explain missing rate sheets, employee defaults, unpriced amounts,
and currency estimates. Excluded factors remain visible and marked Excluded.

## Shared code

`GET /api/wip/my`, `/api/wip/branch`, and `/api/wip/division` return WIP factors
for all selected jobs in one SQL query. Branch accepts an optional `branch` ID;
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
missing-authorization queues. No migration is needed.

UI tests are `npm run test:wip-reports` and `npm run test:wip-reports-browser`.
Browser tests cover ranking, notices, keyboard controls, navigation access,
selection defaults, filter changes, empty states, retry, stale responses, and
mobile layout. Existing WIP and time-summary tests cover the shared code.
