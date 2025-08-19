<script lang="ts">
  import QueryBox from "$lib/components/QueryBox.svelte";
  import { pb } from "$lib/pocketbase";

  let { jobId, startDate, endDate } = $props();

  const formatTwoDecimals = <T,>(v: T) =>
    typeof v === "number" ? (v as number).toFixed(2) : (v as T);

  const tableConfig: any = {
    columnFormatters: {
      value: "dollars" as const,
      hours: formatTwoDecimals,
      meals_hours: formatTwoDecimals,
    },
    omitColumns: ["uid"],
  };
</script>

{#if !startDate || !endDate}
  <div class="px-4">Please select a start and end date.</div>
{:else}
  <div class="px-4">
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
