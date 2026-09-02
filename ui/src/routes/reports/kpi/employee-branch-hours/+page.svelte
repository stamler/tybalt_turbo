<script lang="ts">
  import { resolve } from "$app/paths";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsDateInput from "$lib/components/DSDateInput.svelte";
  import ObjectTable from "$lib/components/ObjectTable.svelte";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import { downloadCSV } from "$lib/utilities";

  type ReportResponse = {
    columns: string[];
    rows: Record<string, string | number>[];
  };

  function formatLocalDate(date: Date) {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
  }

  function previousMonthRange(reference = new Date()) {
    return {
      start: formatLocalDate(new Date(reference.getFullYear(), reference.getMonth() - 1, 1)),
      end: formatLocalDate(new Date(reference.getFullYear(), reference.getMonth(), 0)),
    };
  }

  const initialRange = previousMonthRange();
  let draftStartDate = $state(initialRange.start);
  let draftEndDate = $state(initialRange.end);
  let appliedStartDate = $state(initialRange.start);
  let appliedEndDate = $state(initialRange.end);
  let report = $state<ReportResponse>({ columns: [], rows: [] });
  let loading = $state(false);
  let loaded = $state(false);
  let errorMessage = $state("");
  let initialRequestStarted = $state(false);

  const canView = $derived($globalStore.claims.includes("kpi"));
  const dateError = $derived(
    !draftStartDate || !draftEndDate
      ? "Select both dates."
      : draftStartDate > draftEndDate
        ? "The start date must be on or before the end date."
        : "",
  );

  const tableConfig = $derived({
    columnFormatters: Object.fromEntries(
      report.columns.slice(4).map((column) => [column, formatHours]),
    ),
    omitColumns: [],
    columnOrder: report.columns,
  });

  function reportURL(startDate: string, endDate: string, format = "json") {
    const params = new URLSearchParams({
      start_date: startDate,
      end_date: endDate,
      format,
    });
    return `/api/kpi/reports/employee_branch_hours?${params.toString()}`;
  }

  function formatHours<T>(value: T): string | T {
    if (typeof value !== "number") return value;
    return value.toLocaleString("en-CA", {
      minimumFractionDigits: 0,
      maximumFractionDigits: 2,
    });
  }

  async function loadReport(startDate: string, endDate: string) {
    loading = true;
    errorMessage = "";
    try {
      const response = (await pb.send(reportURL(startDate, endDate), {
        method: "GET",
        requestKey: null,
      })) as ReportResponse;
      report = {
        columns: Array.isArray(response.columns) ? response.columns : [],
        rows: Array.isArray(response.rows) ? response.rows : [],
      };
      loaded = true;
    } catch (error) {
      console.error("Failed to load employee branch hours report:", error);
      errorMessage = "The report could not be loaded. Try again.";
      report = { columns: [], rows: [] };
      loaded = true;
    } finally {
      loading = false;
    }
  }

  async function updateReport() {
    if (dateError) return;
    appliedStartDate = draftStartDate;
    appliedEndDate = draftEndDate;
    await loadReport(appliedStartDate, appliedEndDate);
  }

  async function downloadReport() {
    const url = `${pb.baseUrl}${reportURL(appliedStartDate, appliedEndDate, "csv")}`;
    const filename = `employee_branch_hours_${appliedStartDate}_${appliedEndDate}.csv`;
    await downloadCSV(url, filename);
  }

  $effect(() => {
    if (canView && !initialRequestStarted) {
      initialRequestStarted = true;
      void loadReport(appliedStartDate, appliedEndDate);
    }
  });
</script>

<svelte:head>
  <title>Employee Branch Hours</title>
</svelte:head>

<main class="mx-auto max-w-full p-4 sm:p-6">
  <div class="mb-5">
    <a class="text-sm text-blue-700 hover:underline" href={resolve("/reports/kpi")}>← KPI Reports</a
    >
    <h1 class="mt-1 text-3xl font-bold text-neutral-900">Employee Branch Hours</h1>
    <p class="mt-2 text-neutral-600">
      Regular and training hours assigned to jobs and no-job work, grouped by employee and branch.
    </p>
  </div>

  {#if canView}
    <section class="mb-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 shadow-sm">
      <div class="flex flex-wrap items-end gap-4">
        <label class="grid gap-1 text-sm font-medium text-neutral-700">
          Start date
          <DsDateInput bind:value={draftStartDate} max={draftEndDate || undefined} />
        </label>
        <label class="grid gap-1 text-sm font-medium text-neutral-700">
          End date
          <DsDateInput bind:value={draftEndDate} min={draftStartDate || undefined} />
        </label>
        <DsActionButton
          action={updateReport}
          {loading}
          disabled={!!dateError}
          title="Update report"
          color="blue">Update</DsActionButton
        >
        {#if loaded && !errorMessage}
          <DsActionButton
            action={downloadReport}
            disabled={loading}
            icon="mdi:file-download-outline"
            title="Download CSV"
            color="green"
          />
        {/if}
      </div>
      {#if dateError}
        <p class="mt-2 text-sm text-red-700">{dateError}</p>
      {:else if loaded}
        <p class="mt-2 text-sm text-neutral-500">
          Showing {appliedStartDate} through {appliedEndDate}, inclusive.
        </p>
      {/if}
    </section>

    {#if errorMessage}
      <div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-800">
        {errorMessage}
      </div>
    {:else if loading && !loaded}
      <div class="rounded-lg border border-neutral-200 bg-white p-6 text-neutral-500 shadow-sm">
        Loading report…
      </div>
    {:else if loaded && report.rows.length === 0}
      <div class="rounded-lg border border-neutral-200 bg-white p-6 text-neutral-500 shadow-sm">
        No employees have qualifying hours in this period.
      </div>
    {:else if report.rows.length > 0}
      <section class="report-table rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
        <ObjectTable tableData={report.rows} {tableConfig} />
      </section>
    {/if}
  {:else}
    <div class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-amber-900">
      You need the kpi claim to view this report.
    </div>
  {/if}
</main>

<style>
  .report-table :global(table) {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0;
  }

  .report-table :global(thead) {
    background: #f5f5f5;
  }

  .report-table :global(th),
  .report-table :global(td) {
    padding: 0.65rem 0.75rem;
  }

  .report-table :global(tbody tr:nth-child(even)) {
    background: #fafafa;
  }

  .report-table :global(tbody tr:hover) {
    background: #eff6ff;
  }
</style>
