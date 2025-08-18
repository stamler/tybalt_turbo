<script lang="ts">
  import ObjectTable from "./ObjectTable.svelte";
  import DSActionButton from "./DSActionButton.svelte";

  type TableData = Record<string, any>[];

  let {
    queryName,
    queryValues = [],
    dlFileName,
  } = $props<{
    queryName?: string;
    queryValues?: unknown[];
    dlFileName?: string;
  }>();

  let queryResult = $state<TableData>([]);
  let loading = $state(false);

  function toCsv(rows: TableData): string {
    if (!rows || rows.length === 0) return "";
    const headers = Object.keys(rows[0] ?? {});
    const escape = (val: unknown) => {
      if (val === null || val === undefined) return "";
      const str = String(val);
      const needsQuotes = /[",\n]/.test(str);
      const escaped = str.replace(/"/g, '""');
      return needsQuotes ? `"${escaped}"` : escaped;
    };
    const lines = [headers.join(",")];
    for (const row of rows) {
      lines.push(headers.map((h) => escape(row[h])).join(","));
    }
    return lines.join("\n");
  }

  function download() {
    if (!queryResult || queryResult.length === 0) return;
    const csv = toCsv(queryResult);
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const name = dlFileName ?? "report.csv";
    const a = document.createElement("a");
    a.href = url;
    a.download = name;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  async function runQuery(key: string) {
    loading = true;
    try {
      // TODO: replace with PocketBase-backed endpoint call
      // Example (to be implemented later):
      // const data = await pb.send("/api/reports", { key, values: queryValues });
      // queryResult = data;
      queryResult = [];
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    // Re-run whenever either the query name or values change
    // eslint-disable-next-line @typescript-eslint/no-unused-expressions
    queryValues;
    void runQuery(queryName ?? "");
  });

  const hasResults = $derived(!!queryResult && queryResult.length > 0);
</script>

<div>
  <ObjectTable tableData={queryResult} />
  {#if hasResults}
    <DSActionButton
      title="download report"
      icon="mdi:download"
      color="yellow"
      action={download}
      {loading}
      type="button"
    />
  {/if}
  <!-- Optionally, display a loading indicator outside the button condition -->
  {#if loading}
    <div class="text-sm text-neutral-500">getting data...</div>
  {/if}
</div>
