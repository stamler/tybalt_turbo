<script lang="ts">
  import { resolve } from "$app/paths";
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
  import type { RateSheet } from "./timeSummary";
  let { rateSheet }: { rateSheet?: RateSheet } = $props();
</script>

<TimeSummaryHelp title="How time values are calculated">
  {#if rateSheet?.id}
    <p>
      Rate sheet: <a
        class="text-blue-600 hover:underline"
        href={resolve("/rate-sheets/[id]/details", { id: rateSheet.id })}
        >{rateSheet.name} (rev. {rateSheet.revision})</a
      >.
    </p>
  {/if}
  <p>
    Each entry uses the regular rate for its role from the job's assigned sheet. If no job rate
    matches, an employee default charge-out rate greater than zero is used.
  </p>
  <p>
    Estimated amounts include employee defaults. Partial amounts exclude hours that cannot be
    priced. If no hours can be priced, the value is shown as —.
  </p>
  <p>
    Includes committed and uncommitted entries in the selected dates. Excludes meal hours, time
    amendments, and overtime billing.
  </p>
  <p>
    Shares use the full value for the job and date range. They are unavailable when that value is
    incomplete or zero.
  </p>
  <p>
    Click column headings to sort and table values to filter. The total follows the visible rows;
    the CSV always includes the full date range and all pricing details.
  </p>
  <p>Changes to the assigned sheet, job rates, or employee defaults recalculate past values.</p>
</TimeSummaryHelp>
