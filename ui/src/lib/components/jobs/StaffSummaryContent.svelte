<script lang="ts">
  import QueryBox from "$lib/components/QueryBox.svelte";
  import TimeValuationNote from "./TimeValuationNote.svelte";
  import { formatCurrency } from "$lib/utilities";
  import { pb } from "$lib/pocketbase";

  let { jobId, startDate, endDate, rateSheet } = $props();

  const formatTwoDecimals = <T,>(v: T) =>
    typeof v === "number" ? (v as number).toFixed(2) : (v as T);

  const tableConfig = {
    columnFormatters: {
      value: <T,>(v: T) => (typeof v === "number" ? formatCurrency(v) : "—"),
      total: <T,>(v: T) => (typeof v === "number" ? formatCurrency(v) : "—"),
      percent: <T,>(v: T) => (typeof v === "number" ? `${v.toFixed(1)}%` : "—"),
      estimated_hours: formatTwoDecimals,
      unpriced_hours: formatTwoDecimals,
      total_unpriced_hours: formatTwoDecimals,
      hours: formatTwoDecimals,
      meals_hours: formatTwoDecimals,
    },
    columnLabels: {
      hours: "Hours",
      value: "Known value",
      total: "Known total",
      percent: "Share of value",
      estimated_hours: "Estimated hours",
      unpriced_hours: "Unpriced hours",
      value_status: "Value status",
      total_status: "Total status",
    },
    columnOrder: [
      "given_name",
      "surname",
      "division_code",
      "division_name",
      "hours",
      "estimated_hours",
      "unpriced_hours",
      "value",
      "value_status",
      "total",
      "total_status",
      "percent",
    ],
    omitColumns: [
      "rate_sheet_name",
      "rate_sheet_revision",
      "total_estimated_hours",
      "total_unpriced_hours",
      "uid",
      "number",
      "meals_hours",
    ],
  };
</script>

{#if !startDate || !endDate}
  <div class="px-4">Please select a start and end date.</div>
{:else}
  <div class="px-4">
    <TimeValuationNote {rateSheet} />
    <QueryBox
      queryValues={[jobId, startDate, endDate]}
      {tableConfig}
      dlFileName={`job_${jobId}_staff_summary_${startDate}_${endDate}.csv`}
      fetcher={({ queryValues }) => {
        const [id, start, end] = queryValues as [string, string, string];
        if (!id || !start || !end) return Promise.resolve([]);
        const params = new URLSearchParams({ start_date: start, end_date: end });
        return pb.send(`/api/jobs/${id}/staff/summary?${params.toString()}`, { method: "GET" });
      }}
    />
  </div>
{/if}
