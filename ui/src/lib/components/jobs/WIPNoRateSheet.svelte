<script lang="ts">
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
  import type { JobWIP } from "./wip";
  let { data }: { data: JobWIP } = $props();
</script>

{#if data.no_rate_sheet}
  <TimeSummaryHelp
    label="No rate sheet · employee rates used where available"
    title="Time uses employee defaults"
    warning
  >
    <p>
      This job has no rate sheet. Time uses employee default charge-out rates greater than zero. {data.estimated_hours.toFixed(
        2,
      )} hours use defaults; {data.unpriced_hours.toFixed(2)} hours remain unpriced.
    </p>
  </TimeSummaryHelp>
{/if}
