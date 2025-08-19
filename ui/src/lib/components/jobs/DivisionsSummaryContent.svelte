<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import ObjectTable from "$lib/components/ObjectTable.svelte";

  interface DivisionRow {
    number: string;
    division_code: string;
    division_name: string;
    job_hours: number;
    division_value_dollars: number;
    job_value_dollars: number;
    division_value_percent: number;
  }

  let { jobId, startDate, endDate } = $props();

  let rows: DivisionRow[] = $state([]);
  let loading = $state(false);
  let errorMsg = $state("");

  async function fetchRows() {
    if (!jobId || !startDate || !endDate) return;
    loading = true;
    errorMsg = "";
    try {
      const params = new URLSearchParams({ start_date: startDate, end_date: endDate });
      const url = `/api/jobs/${jobId}/divisions/summary?${params.toString()}`;
      const res: DivisionRow[] = await pb.send(url, { method: "GET" });
      rows = Array.isArray(res) ? res : [];
    } catch (err) {
      console.error("Failed to fetch divisions summary", err);
      errorMsg = "Failed to load divisions summary";
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    fetchRows();
  });

  const formatTwoDecimals = <T,>(v: T) =>
    typeof v === "number" ? (v as number).toFixed(2) : (v as T);

  const tableConfig: any = {
    columnFormatters: {
      division_value_dollars: "dollars" as const,
      job_value_dollars: "dollars" as const,
      division_value_percent: "percent" as const,
      job_hours: formatTwoDecimals,
    },
    omitColumns: [],
  };
</script>

{#if !startDate || !endDate}
  <div class="px-4">Please select a start and end date.</div>
{:else if loading}
  <div class="px-4">Loading…</div>
{:else if errorMsg}
  <div class="px-4 text-red-700">{errorMsg}</div>
{:else}
  <div class="px-4">
    <ObjectTable tableData={rows} {tableConfig} />
  </div>
{/if}
