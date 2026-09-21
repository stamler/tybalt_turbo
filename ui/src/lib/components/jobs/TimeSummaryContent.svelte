<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import TimeSummaryTable from "./TimeSummaryTable.svelte";
  import type { RateSheet, SummaryKind, TimeSummaryRow } from "./timeSummary";

  let {
    jobId,
    startDate,
    endDate,
    rateSheet,
    kind,
  }: {
    jobId: string;
    startDate: string;
    endDate: string;
    rateSheet?: RateSheet;
    kind: SummaryKind;
  } = $props();
  const requestId = $props.id();
  let rows = $state<TimeSummaryRow[]>([]);
  let loading = $state(false);
  let error = $state(false);

  $effect(() => {
    const id = jobId,
      start = startDate,
      end = endDate,
      summaryKind = kind;
    rows = [];
    error = false;
    loading = false;
    if (!id || !start || !end) return;
    const controller = new AbortController();
    let active = true;
    loading = true;
    const params = new URLSearchParams({ start_date: start, end_date: end });
    // Ignore an older response after a date or job change. Never leave values
    // from the previous range visible while the new summary is loading.
    pb.send<TimeSummaryRow[]>(`/api/jobs/${id}/${summaryKind}/summary?${params}`, {
      method: "GET",
      signal: controller.signal,
      requestKey: `${requestId}:${rateSheet?.id ?? ""}`,
    })
      .then((data) => {
        if (active) rows = data;
      })
      .catch(() => {
        if (active) error = true;
      })
      .finally(() => {
        if (active) loading = false;
      });
    return () => {
      active = false;
      controller.abort();
    };
  });
</script>

<div class="px-4">
  {#if !startDate || !endDate}
    <p>Please select a start and end date.</p>
  {:else if loading}
    <p role="status" class="py-3 text-neutral-600">Loading time summary…</p>
  {:else if error}
    <p role="alert" class="py-3 text-red-700">
      Failed to load the time summary. Change the date range or reload to try again.
    </p>
  {:else}
    <TimeSummaryTable
      {rows}
      {kind}
      {rateSheet}
      filename={`job_${jobId}_${kind}_summary_${startDate}_${endDate}.csv`}
    />
  {/if}
</div>
