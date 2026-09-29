<script lang="ts">
  import WIPTimeDetails from "./WIPTimeDetails.svelte";
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
  import type { JobWIP } from "./wip";
  let { data, compact = false }: { data: JobWIP; compact?: boolean } = $props();
</script>

{#if data.no_rate_sheet}
  <TimeSummaryHelp
    label="No rate sheet · employee rates used where available"
    title="Time uses employee defaults"
    warning
    symbol="NR"
    iconOnly={compact}
  >
    <WIPTimeDetails {data} />
  </TimeSummaryHelp>
{/if}
