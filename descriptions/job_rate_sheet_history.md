# Job rate sheet history

Status: Proposed
Last updated: 2026-10-15

## Current behaviour

A project has one rate sheet, `jobs.rate_sheet`. It is required when the
project is created and must be active at that time. Proposals have no rate
sheet.

Changing the rate sheet of an existing project is possible, but strongly
discouraged. `validateRateSheetIsActive` in `app/hooks/jobs.go` allows it only
when all of these hold:

- The user is active and holds the `job` claim, as for any project edit.
- The user holds the `rate_sheet_revise` claim. Without it, the save fails
  with `missing_claim`.
- The new sheet is active.
- The sheet is replaced, not cleared. A rate sheet cannot be removed once set.

As of 2026-10-15, no users hold the `rate_sheet_revise` claim. In practice,
nobody can change the rate sheet of an existing project.

The job editor shows the rate sheet selector to every user who can edit the
job. Users without `rate_sheet_revise` learn that they cannot change it only
when the save fails.

### Why changing it is discouraged

Time entries store hours and role, but no rate. `job_priced_time_entries.sql`
prices every entry with the job's current sheet when a report runs. A change
therefore reprices all past time on the job, not only time from the change
onward. WIP, WIP reports, the Staff and Divisions summaries, and the
missing-rates breakdown all show the new values. The app does not keep the
previous values, so they cannot be recovered or compared.

Until this proposal is implemented, `rate_sheet_revise` holders should change a
project's rate sheet only to correct a sheet that was wrong from the start. In
that case, repricing all past time is the intended result. A rate change that
applies only from a date, such as a client accepting a new revision, cannot be
recorded correctly today.

## Problem

Rate sheet revisions already have `rate_sheets.effective_date`, and activating a
revision deactivates the others with the same name. That date says when a
revision became available. It does not say when a job moved to it. A client may
accept a new revision months after it was issued, or never.

So a project cannot move to a new sheet from a given date. The only choices are
to keep the old sheet for all time, or to switch and reprice all past time with
the new one.

## Goal

Record which rate sheet applies to a project from which date. Price each time
entry with the sheet in effect on the entry's date. Keep the history editable
by `rate_sheet_revise` holders so a wrong assignment can be corrected.

Expense allowances already work this way: the `expense_allowance_totals` view joins
the `expense_rates` row with the latest `effective_date` on or before the
expense date.

## Data model

New collection `job_rate_sheets`:

| Field            | Type                                | Notes                                                              |
| ---------------- | ----------------------------------- | ------------------------------------------------------------------ |
| `job`            | relation to `jobs`, required        | Projects only. Cascade delete.                                     |
| `rate_sheet`     | relation to `rate_sheets`, required |                                                                    |
| `effective_date` | text `YYYY-MM-DD`                   | Empty only on the opening row.                                     |
| `reason`         | text                                | Required on every row except the opening row created with the job. |
| `created_by`     | relation to `users`                 | Set by the server.                                                 |
| `updated_by`     | relation to `users`                 | Set by the server.                                                 |

`effective_date` uses the same text format as `time_entries.date` and
`rate_sheets.effective_date`, so string comparison gives date order.

Indexes:

- unique `(job, effective_date)`
- unique partial index on `job` where `effective_date = ''`, so a project has
  at most one opening row

### Row rules

- A project with a rate sheet has exactly one opening row. It covers all time
  before the first dated row, so no entry falls outside the history.
- A dated row applies from its `effective_date`, inclusive, until the next
  dated row.
- Effective dates cannot be in the future. A future date would make
  `jobs.rate_sheet` stop meaning "current" without any write to the job.
- A dated row's date must be on or after its rate sheet revision's
  `effective_date`. A revision cannot apply before it existed. The opening row
  is exempt because it covers all earlier time.
- The rate sheet must be active when a row is created or its `rate_sheet`
  changes. Rows keep inactive sheets after deactivation, as `jobs.rate_sheet`
  does today. Changing only the date or reason does not require an active sheet.
- Two consecutive rows must not use the same sheet. Such a row adds nothing,
  and rejecting it keeps the history readable.
- The opening row cannot be deleted. Changing its sheet replaces today's
  "change the job's rate sheet" and restates all time up to the first dated row.
- Proposals have no rows. `cleanJob` already clears their rate sheet.

### `jobs.rate_sheet`

Keep the field. A hook keeps it equal to the sheet on the latest row (the
latest `effective_date`, or the opening row if there are no dated rows). It is
written in the same transaction as the history change.

This keeps these working unchanged:

- the legacy writeback in `jobs_writeback.go`, which sends one sheet per job
- the `no_rate_sheet` flag in `job_wip.sql`
- job lists, details, and the job request views that show the current sheet

Direct writes to `jobs.rate_sheet` on existing projects are rejected. The field
changes only through the history action, using a new workflow mutation in the
same way that `ProjectWorkflowMutationChangeClient` guards `client`.
`validateRateSheetIsActive` keeps its create-time checks. Its update path,
which requires `rate_sheet_revise` and forbids clearing, moves to the history
action.

## Permissions

- **View:** anyone who can view the job.
- **Create, edit, or delete a row:** an active user with both the `job` and
  `rate_sheet_revise` claims, matching today's rule for changing a project's
  rate sheet.
- **Opening row on a new project:** created automatically. Job request approval
  (`ProjectWorkflowMutationCreateJob` in `job_requests.go`) creates it in the
  same transaction as the job, with no `rate_sheet_revise` check. Projects
  created directly when job request approval is off do the same.

All writes go through a custom route, not PocketBase's record API, so the job
mirror, the row rules, and the claim checks run in one transaction. The
collection's create, update, and delete rules are null.

## Pricing

Only `job_priced_time_entries.sql` changes. Every time valuation already uses
it: `job_wip_api.go`, `wip_reports.go`, `job_staff_summary_api.go`,
`job_divisions_summary_api.go`, and `job_wip_missing_rates.go`.

Turn the history into date ranges for the selected jobs, then join each entry
to the range that contains its date:

```sql
, rate_periods AS (
  SELECT jrs.job, jrs.rate_sheet,
    jrs.effective_date AS starts,
    LEAD(jrs.effective_date) OVER (
      PARTITION BY jrs.job ORDER BY jrs.effective_date
    ) AS ends
  FROM selected_jobs j
  JOIN job_rate_sheets jrs ON jrs.job = j.id
)
```

The opening row's empty date sorts first. Then:

```sql
LEFT JOIN rate_periods p
  ON p.job = te.job
  AND te.date >= p.starts
  AND (p.ends IS NULL OR te.date < p.ends)
LEFT JOIN rate_sheets rs ON rs.id = p.rate_sheet
LEFT JOIN rate_sheet_entries rse
  ON rse.rate_sheet = p.rate_sheet AND rse.role = te.role
```

`te.date >= ''` is always true, so the opening row needs no special case. The
employee default charge-out rate and the Estimated, Partial, and unpriced rules
stay as they are.

### Performance

A project has only a few rows, and the `(job, effective_date)` index covers
the scan. WIP reports load hundreds of jobs at once, so repeat the measurements
in `wip_performance.md` before and after, on the same production snapshot.

## API and UI

### Time valuation

`JobTimeValuation` returns one `rate_sheet_name` and `rate_sheet_revision`.
`TimeSummaryTable.svelte` reads `rows[0].rate_sheet_revision === null` to
decide that a job has no sheet. With history, a job's time can be priced by
several sheets.

- Replace the single name and revision with a `rate_sheets` list of
  `{ name, revision, starts, ends }` for the periods that contain priced time
  in the report range.
- Use `jobs.rate_sheet` for the "no rate sheet" check rather than inferring it
  from row data.
- When more than one sheet contributed, the summaries say so and list the
  periods.

### Missing rates

The breakdown groups missing-role-rate hours by sheet. A role can be priced in
one period and not in the next.

### Job editor and details

- On existing projects the rate sheet selector is read-only, and shows the
  current sheet.
- Job details gets a Rate sheets section listing the history: sheet name and
  revision, effective date ("from the start" for the opening row), reason, and
  who set it.
- `rate_sheet_revise` holders see Add, Edit, and Delete actions on that list.
  The add form defaults the date to today and lists only active sheets.
- Warn before saving a dated row earlier than today: past time values will
  change.

## Migration

1. Create `job_rate_sheets` and its indexes.
2. Backfill one opening row for every project with a non-empty `rate_sheet`,
   including imported jobs. Leave `reason` empty and `created_by` unset on
   backfilled rows.
3. Projects with no rate sheet get no rows. Their time remains priced only by
   employee defaults, as today.

The backfill produces identical prices: every project has one range covering
all time, using its current sheet. Run the WIP report and a sample of job
summaries before and after the migration and compare the results.

## Tests

- **Pricing:** entries before, on, and after a dated row; several dated rows;
  an inactive sheet in history; a role present in one sheet and missing from
  the next; a project with no rows.
- **Row rules:** second opening row; duplicate date; future date; date before
  the revision's `effective_date`; consecutive duplicate sheet; deleting the
  opening row; inactive sheet on create; date-only edit of a row whose sheet is
  now inactive.
- **Mirror:** `jobs.rate_sheet` follows the latest row after create, edit,
  and delete. Direct writes to it are rejected on existing projects.
- **Permissions:** missing `job` or `rate_sheet_revise`; inactive user; the
  opening row created through job request approval without
  `rate_sheet_revise`.
- **Migration:** WIP values are identical before and after the backfill.

## Open questions

- **Invoiced periods:** when invoiced amounts are tracked
  (`WIP_invoiced_amount_tracking.md`), should a dated row earlier than the last
  invoice be blocked? For now, the warning and the required reason are the
  only safeguards.
- **Legacy system:** the writeback sends only the current sheet. If the legacy
  system prices time itself, it will price all time with that sheet. Is that
  acceptable while both systems run?
- **Revision rollout:** when a new revision is activated, should the app offer
  to add a dated row to every project on the previous revision, effective from
  the revision's date? Today each project has to be moved by hand.
- **Overtime:** `rate_sheet_entries.overtime_rate` is not used for pricing.
  This proposal does not change that.
