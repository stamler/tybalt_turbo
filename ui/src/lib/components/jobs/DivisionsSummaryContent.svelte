<script lang="ts">
  import TimeSummaryContent from "./TimeSummaryContent.svelte";
  import type { RateSheet } from "./timeSummary";
  let {
    jobId,
    startDate,
    endDate,
    rateSheet,
  }: {
    jobId: string;
    startDate: string;
    endDate: string;
    rateSheet?: RateSheet;
  } = $props();
</script>

<TimeSummaryContent {jobId} {startDate} {endDate} {rateSheet} kind="divisions" />
