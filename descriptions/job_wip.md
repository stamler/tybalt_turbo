# Job WIP

The WIP tab follows Active POs on Job Details. It shows the included value as a
percentage of `jobs.project_value`. This is a comparison of charge-out values
and commitments with project value, not a measure of physical completion,
actual labour cost, cash paid, or invoiced revenue.

## Calculation

`GET /api/jobs/{id}/wip` requires the same user authentication as the other job
summary routes. It returns 404 for an unknown job. SQL returns each factor in
CAD, with counts for estimates and amounts that cannot be valued. The browser
adds the selected factors and calculates percentages without another request.

```
included value = time value + included expenses + included remaining active POs
percentage = included value / project value * 100
```

The report covers only the selected job, not its parent or child jobs. It is a
current report, not a historical snapshot. The report date is today in UTC.

### Time

Use the shared `job_priced_time_entries.sql` query through the report date.
Include committed and uncommitted time, using the assigned rate-sheet revision
and recorded role. If no job rate matches, use a positive employee default
charge-out rate. Exclude meal hours, time amendments, and overtime billing.
Rate changes can change past time values. Retain unpriced hours and mark their
value Partial. Employee defaults make the value Estimated.

### Expenses

Include job expenses with a commitment timestamp, no rejection timestamp, and
an expense date through the report date. Exclude draft, pending, and rejected
expenses. Use positive `settled_total` amounts in CAD. For CAD or legacy blank
currencies, fall back to `total`. A nonzero foreign expense without settlement
is unpriced; never treat its native amount as CAD.

### Remaining active POs

Include POs whose current status is Active and whose job is this job. Include
future commitments. Closed, Cancelled, and Unapproved POs contribute nothing.

- One-Time and Cumulative: use `total` as the authorised amount.
- Recurring: use `approval_total`, which covers all occurrences. A positive
  per-occurrence total with a missing full approval value is unpriced.
- Subtract committed, non-rejected expenses through the report date from each
  PO, in its native currency. PO and expense currencies must match on save.
- Clamp each remaining native balance to zero before adding POs together.
- Convert foreign balances at the current currency rate and round each to
  cents. Mark these balances Estimated. A positive balance with no exchange
  rate is unpriced. CAD and legacy blank currencies use a rate of one.

The expense deduction always applies, including when Include committed expenses
is off.
Thus a committed expense appears in Expenses and reduces the PO commitment,
rather than being counted twice. Actual CAD settlement can differ from the
current exchange-rate estimate used for the remaining commitment.

## Display and controls

Time value is always included. Include committed expenses and Include active POs
start checked. A question-mark button beside Time value explains: "Hours × job
rate, with employee defaults where needed."
Excluded rows remain visible with an Excluded label and no project percentage.
The total, remaining project value, budget bar, and optional doughnut update
when a checkbox changes. Opening the tab loads fresh data and resets controls.

The table and stacked bar show percentages of project value. The bar uses blue
for time, teal for expenses, amber for POs, and grey for unused project value.
A black marker shows 100% of project value. If the total exceeds project value,
the marker moves inside the bar and the amount over project value is shown.
The percentage is not capped at 100.

Show breakdown chart reveals a doughnut with shares of the included total.
Its caption states that denominator. Unchecked components have no slice.
The table and chart legend provide text values as well as colours.

A project value of zero or less hides project percentages and the budget bar;
the values and composition chart remain available. An empty report has zero
values and no doughnut. Signed corrections remain in totals but suppress the
charts because they cannot be drawn as positive segments.

If an included factor is Partial, show its known value and hide its percentage.
Hide the overall percentage, remaining project value, and charts until all
included factors can be valued. Complete factors still show their own project
percentages. Excluding an incomplete expense or PO factor restores the overall
percentage. A Partial zero value is shown as a dash. Estimated values remain
included in percentages and charts.

Question-mark controls explain the rules and exceptions. A no-rate-sheet notice
explains employee defaults. Loading failures have a retry button, and cancelled
or late requests cannot replace the current job's results.

## Validation

Backend coverage is in `app/job_wip_test.go`, with dedicated append-only CSV
fixtures. The fixtures cover PO types and states, expense commitment states,
foreign settlement and conversion, overdrawn POs, future dates, empty jobs,
missing values, authentication, and missing jobs. No schema migration is needed.

From `ui/`, run `npm run test:wip` and `npm run test:wip-browser`. The browser
checks use real components with fixture API responses. They share the browser
harness used by `test:time-summary-browser`. An installed Playwright and browser
are required; `PLAYWRIGHT_MODULE` selects an external module and
`PLAYWRIGHT_CHANNEL=chrome` selects installed Chrome.

This feature is separate from the draft invoice-entry proposal in
`WIP_invoiced_amount_tracking.md`.
