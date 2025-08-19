<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import ObjectTable from "$lib/components/ObjectTable.svelte";

  interface StaffRow {
    number: string;
    given_name: string;
    surname: string;
    hours: number;
    value: number;
    meals_hours: number;
    uid: string;
  }

  let { jobId, startDate, endDate } = $props();

  let rows: StaffRow[] = $state([]);
  let loading = $state(false);
  let errorMsg = $state("");

  async function fetchRows() {
    if (!jobId || !startDate || !endDate) return;
    loading = true;
    errorMsg = "";
    try {
      const params = new URLSearchParams({ start_date: startDate, end_date: endDate });
      const url = `/api/jobs/${jobId}/staff/summary?${params.toString()}`;
      const res: StaffRow[] = await pb.send(url, { method: "GET" });
      rows = Array.isArray(res) ? res : [];
    } catch (err) {
      console.error("Failed to fetch staff summary", err);
      errorMsg = "Failed to load staff summary";
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    fetchRows();
  });

  const formatTwoDecimals = <T,>(v: T) => (typeof v === "number" ? (v as number).toFixed(2) : (v as T));

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
{:else if loading}
  <div class="px-4">Loading…</div>
{:else if errorMsg}
  <div class="px-4 text-red-700">{errorMsg}</div>
{:else}
  <div class="px-4">
    <ObjectTable tableData={rows} tableConfig={tableConfig} />
  </div>
{/if}


