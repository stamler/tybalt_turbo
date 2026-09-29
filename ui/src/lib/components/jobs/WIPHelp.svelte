<script lang="ts">
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
</script>

<TimeSummaryHelp title="How WIP is calculated">
  <p>
    WIP compares recorded time, included expenses, and remaining active PO commitments with the
    project value. It does not measure work completed or cash paid.
  </p>
  <p>
    Time uses the same job rates and employee defaults as the Staff and Divisions summaries,
    including committed and uncommitted entries. Meal hours, amendments, and overtime billing are
    excluded.
  </p>
  <p>
    Expenses include committed, non-rejected records through the report date. Foreign expenses use
    settled CAD amounts.
  </p>
  <p>
    Active POs include future commitments. Their remaining value subtracts committed expenses, even
    when Include committed expenses is off. Recurring POs use the full approved value across all
    occurrences. Each remaining balance is at least zero.
  </p>
  <p>
    Foreign PO balances are converted using the current exchange rate, so they are estimates.
    Amounts with missing settlement, exchange rates, or recurring approval values are marked
    Partial.
  </p>
  <p>
    The percentage is (time value + selected expenses + selected remaining active POs) ÷ project
    value × 100. Remaining is project value minus that same selected total. Negative balances show
    amounts over the project value. The checkboxes select which amounts to include. The optional Job
    Details chart shows shares of that total. Each job’s WIP excludes its parent and child jobs.
  </p>
</TimeSummaryHelp>
