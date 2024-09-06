<script lang="ts">
  import { onMount } from "svelte";
  import _ from "lodash";
  import Icon from "@iconify/svelte";
  import { formatDollars, formatPercent } from "$lib/utilities";
  // import DownloadData from "@/components/DownloadData.svelte";

  export let tableData: Record<string, any>[] = [];
  export let tableConfig: {
    columnFormatters: Record<string, (<T>(value: T) => string | T) | "dollars" | "percent">;
    omitColumns?: string[];
  } = { columnFormatters: {}, omitColumns: [] };

  let internalTableData: Record<string, any>[] = [];
  let filters: { key: string; value: string }[] = [];
  let sortColumn = "";
  let order = 0;
  let instanceId: string;

  $: columns =
    internalTableData.length > 0
      ? Object.keys(internalTableData[0]).filter(
          (col) => col !== `_idx_${instanceId}` && !tableConfig.omitColumns.includes(col),
        )
      : [];

  $: columnAlignments =
    internalTableData.length > 0
      ? columns.reduce((acc, col) => {
          acc[col] = typeof internalTableData[0][col] === "number" ? "text-right" : "text-left";
          return acc;
        }, {})
      : {};

  $: columnFormatters = Object.entries(tableConfig.columnFormatters).reduce(
    (acc, [col, formatter]) => {
      if (formatter === "dollars") acc[col] = formatDollars;
      else if (formatter === "percent") acc[col] = formatPercent;
      else acc[col] = formatter;
      return acc;
    },
    {},
  );

  function indexData(data: Record<string, any>[]) {
    if (data.length > 0 && data[0].id !== undefined) return data;
    return data.map((row, idx) => ({ ...row, [`_idx_${instanceId}`]: idx }));
  }

  function flattenData(data: Record<string, any>[]) {
    return data.map((row) => {
      const flatRow = { ...row };
      Object.entries(flatRow).forEach(([key, value]) => {
        if (value === null) return;
        if (typeof value === "object") {
          Object.entries(value).forEach(([nestedKey, nestedValue]) => {
            flatRow[`${key}_${nestedKey}`] = nestedValue;
          });
          delete flatRow[key];
        }
      });
      return flatRow;
    });
  }

  function formatCell(columnName: string, value: any): string | any {
    return columnFormatters[columnName] ? columnFormatters[columnName](value) : value;
  }

  function sort(column: string) {
    sortColumn = column;
    order = (order + 1) % 3;
    updateInternalTableData();
  }

  function removeFilter(filter: { key: string; value: string }) {
    filters = filters.filter((f) => f !== filter);
    updateInternalTableData();
  }

  function addFilter(key: string, value: string) {
    filters = [...filters, { key, value }];
    updateInternalTableData();
  }

  function updateInternalTableData() {
    let data = flattenData(indexData(tableData));
    data = data.filter((row) => filters.every((filter) => row[filter.key] === filter.value));
    if (order !== 0) {
      data = _.orderBy(data, sortColumn, order === 1 ? "asc" : "desc");
    }
    internalTableData = data;
  }

  onMount(() => {
    instanceId = Math.random().toString(36).substring(7);
    updateInternalTableData();
  });

  $: {
    if (tableData) {
      updateInternalTableData();
    }
  }
</script>

{#if Array.isArray(internalTableData) && internalTableData.length > 0}
  <!-- <DownloadData data={internalTableData} {columns} dlFileName="report.csv" /> -->
  <div class="flex flex-wrap">
    {#each filters as f (f.key)}
      <div class="mb-2 mr-2 flex rounded-sm border border-blue-700 bg-blue-200 px-2 py-1">
        <Icon icon="mdi:close-circle" class="w-5 pr-1" on:click={() => removeFilter(f)} />
        {f.key}: {f.value}
      </div>
    {/each}
  </div>
  <div id="tablecontainer">
    {#if internalTableData.length > 0}
      <table>
        <thead>
          <tr>
            {#each columns as col}
              <th class="align-bottom">
                <a
                  href="#"
                  class="hover:underline"
                  title="sort"
                  on:click|preventDefault={() => sort(col)}
                >
                  {col}
                </a>
                <span class="inline-block h-5 w-4">
                  {#if sortColumn === col}
                    {#if order === 2}
                      <Icon icon="mdi:sort-descending" class="w-5" />
                    {:else if order === 1}
                      <Icon icon="mdi:sort-ascending" class="w-5" />
                    {/if}
                  {/if}
                </span>
              </th>
            {/each}
            {#if $$slots.rowActions}
              <th></th>
            {/if}
          </tr>
        </thead>
        <tbody>
          {#each internalTableData as row (row.id)}
            <tr>
              {#each columns as col}
                <td class="pr-4" class:text-right={columnAlignments[col] === "text-right"}>
                  <a
                    href="#"
                    class="hover:underline"
                    on:click|preventDefault={() => addFilter(col, row[col])}
                  >
                    {formatCell(col, row[col])}
                  </a>
                </td>
              {/each}
              {#if $$slots.rowActions}
                <td>
                  <slot name="rowActions" id={row.id} idx={row[`_idx_${instanceId}`]} />
                </td>
              {/if}
            </tr>
          {/each}
        </tbody>
      </table>
    {:else}
      <span>No Data</span>
    {/if}
  </div>
{:else}
  <div>No Data</div>
{/if}

<style>
  #tablecontainer {
    overflow-x: auto;
    white-space: nowrap;
  }
  thead th {
    padding-right: 1em;
    border-bottom: 1px solid grey;
  }
</style>
