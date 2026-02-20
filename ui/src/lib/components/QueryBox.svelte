<script lang="ts">
  import ObjectTable from "./ObjectTable.svelte";
  import DSActionButton from "./DSActionButton.svelte";
  import type { BaseSystemFields } from "$lib/pocketbase-types";
  import { downloadCsvRows } from "$lib/utilities";

  type TableData = Record<string, any>[];

  type TableConfig = {
    columnFormatters: Record<string, (<T>(value: T) => string | T) | "dollars" | "percent">;
    omitColumns?: string[];
  };

  let {
    queryValues = [],
    dlFileName,
    tableConfig,
    fetcher,
    auto = true,
    deps,
  } = $props<{
    queryValues?: unknown[];
    dlFileName?: string;
    tableConfig?: TableConfig;
    fetcher: (args: { queryValues?: unknown[] }) => Promise<TableData>;
    auto?: boolean;
    deps?: unknown[]; // optional explicit dependencies for re-running
  }>();

  // Optional: pass ObjectTable config as the last element of queryValues
  let tableConfigResolved = $state<TableConfig>({ columnFormatters: {}, omitColumns: [] });
  $effect(() => {
    tableConfigResolved = tableConfig ?? { columnFormatters: {}, omitColumns: [] };
  });

  let queryResult = $state<TableData>([]);
  let loading = $state(false);
  let errorMsg = $state("");

  function download() {
    if (!queryResult || queryResult.length === 0) return;
    const name = dlFileName ?? "report.csv";
    downloadCsvRows(name, queryResult);
  }

  async function runQuery() {
    loading = true;
    try {
      errorMsg = "";
      try {
        const data = await fetcher({ queryValues });
        queryResult = Array.isArray(data) ? data : [];
      } catch (err) {
        console.error("QueryBox fetcher error", err);
        errorMsg = "Failed to load data";
        queryResult = [];
      }
    } finally {
      loading = false;
    }
  }

  const triggerKey = $derived(JSON.stringify(deps ?? queryValues ?? []));
  $effect(() => {
    // Re-run whenever dependencies or queryValues change
    triggerKey;
    if (auto) {
      void runQuery();
    }
  });

  const hasResults = $derived(!!queryResult && queryResult.length > 0);
</script>

<div>
  <ObjectTable
    tableData={queryResult as Record<string, any>[] & BaseSystemFields<any>[]}
    tableConfig={tableConfigResolved}
  />
  {#if hasResults}
    <DSActionButton
      title="download report"
      icon="mdi:download"
      color="yellow"
      action={download}
      {loading}
      type="button"
    />
  {:else if errorMsg}
    <div class="text-sm text-red-700">{errorMsg}</div>
  {/if}
  <!-- Optionally, display a loading indicator outside the button condition -->
  {#if loading}
    <div class="text-sm text-neutral-500">getting data...</div>
  {/if}
</div>
