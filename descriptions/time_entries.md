# Time Entries

This note documents time-entry processing and job summary values.

## Job Summary Values

The Staff summary and Divisions summary on Job Details calculate values in SQL.
Each time entry uses the regular rate for its recorded `role` from the exact
`rate_sheet` revision assigned to its job. The calculation is `hours * rate`.
If the job has no rate sheet, the entry has no role, or no rate matches that
role, use the employee's `admin_profiles.default_charge_out_rate` when it is
greater than zero. Entries are priced before they are grouped by employee or
division. A matching job rate always takes priority over the employee default.

Both summaries include committed and uncommitted time entries within the
selected dates, including both date limits. They exclude time amendments.
Meal hours are not priced. Overtime rates are not used because time entries
do not identify which hours attract an overtime billing rate.

An inactive sheet remains valid for a job that already uses it. The summaries
do not select revisions by entry date or switch to the latest active revision.
Changes to the assigned sheet, its rates, or employee default rates recalculate
past values; these summaries do not preserve historical billing values.

Hours priced with employee defaults are reported as `estimated_hours`. Hours
with neither a matching job rate nor a positive employee default remain in the
hours total and are reported as `unpriced_hours`. `value` and `total` include
estimates but exclude unpriced hours. `value_status` and `total_status` are:

- `Rate sheet`: no hours use an employee default or remain unpriced.
- `Estimated`: some hours use an employee default, and no hours remain unpriced.
- `Incomplete`: some hours remain unpriced, even if other hours use defaults.

Percentages are available for complete estimates. They are null when the total
is incomplete or zero. A range without entries returns an empty array.
Zero-hour entries do not make a total estimated or incomplete.

CSV exports include the rate sheet name and revision, estimated and unpriced
hours, and pricing status. The shared query in `app/routes/job_priced_time_entries.sql`
keeps the pricing and entry selection rules the same for both summaries.

### Summary Display

Both summaries show four columns: Staff member or Division, Hours, Value, and
Share. Staff names and division codes/names share one cell. A single total row
replaces the repeated total column.

The question-mark button opens the calculation rules and rate-sheet details.
Normal amounts have no status label. Affected amounts have a clickable
`Estimated` label when employee defaults are used, or `Partial` when some hours
cannot be priced. The explanation states how many hours are affected. Partial
takes priority when a row contains both estimated and unpriced hours. If all
nonzero hours in a row are unpriced, its value is shown as `—`, not `$0.00`.

When no rate sheet is assigned, one notice above the table explains employee
defaults. Repeated Estimated labels are hidden in this case; Partial labels
remain visible. The total follows the same rules and is labelled Known total
when it is incomplete.

Column headings sort the table. Clicking a table value filters it; clicking a
filter removes it. With filters active, the footer shows the visible total.
Shares still use the full job/date-range value as the denominator. CSV exports
always contain all API rows for that date range, including pricing metadata
and status fields hidden from the table.

Help panels support keyboard and touch use, Escape, an explicit close button,
and clicks outside the panel. They appear above the table's scroll area so
horizontal scrolling cannot clip them. A date or job change clears old results;
late responses cannot replace the current range. Load failures are shown
separately from empty results.

UI checks run with `npm run test:time-summary` and
`npm run test:time-summary-browser` from `ui/`. The browser checks use fixture
responses and real components in Chromium. They require Playwright and an
installed browser. Set `PLAYWRIGHT_MODULE` to use an external Playwright module
and `PLAYWRIGHT_CHANNEL=chrome` to use an installed Chrome browser.

The [Job WIP report](job_wip.md) also uses the shared time pricing query. It
combines time values with optional committed expenses and remaining active POs.

## Branch Resolution

`time_entries.branch` now follows the same precedence as `purchase_orders`:

1. If `job` is set, `branch` is forced to the selected job's branch.
2. If `job` is blank and `branch` is blank, `branch` defaults from the user's
   `admin_profiles.default_branch`.
3. If `job` is blank and `branch` is already set, that explicit branch is
   preserved.

This means time entries no longer always derive branch from
`admin_profiles.default_branch`.

## Related Behavior

- When a job is present, the job must exist and be valid for time tracking.
- When a job is present, `role` is required.
- When a job is present, the selected division must be allocated to that job.
- If a referenced job exists but has no `branch`, the create/update is rejected.
- Branch claim enforcement applies to the resolved branch value.

## UI Exposure of Rule (3)

Rule (3) — preserving an explicit non-default branch on a jobless time entry —
is currently **only reachable via the API**. The SvelteKit time-entry editor
does **not** expose a branch picker by default.

The picker exists in `ui/src/lib/components/TimeEntriesEditor.svelte` but is
gated by an in-code feature flag:

```ts
const EXPLICIT_BRANCH_PICKER_ENABLED = false;
```

While the flag is `false`:

- the editor never renders a branch field
- the editor never sends a `branch` value in create/update payloads
- the backend therefore always falls through to rule (2) (default branch)
  for entries created via the UI

When the flag is flipped to `true`, the picker mirrors the
`PurchaseOrdersEditor` behavior: visible only when no job is selected,
filtered by `branches.allowed_claims`, defaulting to the caller's default
branch, and "pinned" once the user manually changes it.

The flag is intentionally code-only (no env var, no `appConfig` toggle, no
user claim) so that enabling it requires a code review and a merged PR. This
is to give stakeholders time to discuss whether jobless cross-branch time
entries are a workflow we want to surface to end users at all.

## Copy To Tomorrow

`POST /api/time_entries/{id}/copy_to_tomorrow` copies the source entry's schema
fields into a new record and then runs normal time-entry processing. As a
result:

- copied entries with a job resolve branch from the job
- copied entries without a job preserve an explicit branch from the source entry
- copied entries without a job and without a branch fall back to the user's
  default branch
