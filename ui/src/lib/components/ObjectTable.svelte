<script lang="ts" generics="T extends BaseSystemFields<any>">
  import type { Snippet } from "svelte";
  import type { BaseSystemFields } from "$lib/pocketbase-types";
  import { onMount } from "svelte";
  import _ from "lodash";
  import Icon from "@iconify/svelte";
  import { formatDollars, formatPercent } from "$lib/utilities";
  // import DownloadData from "@/components/DownloadData.svelte";

  type TableData = Record<string, any>[]; // we do not use T[] because there may be data that does not have an id
  interface TableConfig {
    // a function applied to each value in the specified column to format it
    columnFormatters: Record<string, (<T>(value: T) => string | T) | "dollars" | "percent">;
    omitColumns?: string[];
  }

  let {
    tableData = [],
    tableConfig = { columnFormatters: {}, omitColumns: [] },
    rowActions,
  } = $props<{
    tableData: TableData;
    tableConfig?: TableConfig;
    rowActions?: Snippet<[string, string]>;
  }>();

  let internalTableData = $state<Record<string, any>[]>([]);
  let filters = $state<{ key: string; value: string }[]>([]);
  let sortColumn = $state("");
  let order = $state(0);
  let instanceId = $state("");
  let idVariableName = $state("id");

  let columns = $derived(
    internalTableData.length > 0
      ? Object.keys(internalTableData[0]).filter(
          (col) => col !== `_idx_${instanceId}` && !tableConfig.omitColumns.includes(col),
        )
      : [],
  );

  let columnAlignments = $derived(
    internalTableData.length > 0
      ? columns.reduce(
          (acc, col) => {
            acc[col] = typeof internalTableData[0][col] === "number" ? "text-right" : "text-left";
            return acc;
          },
          {} as Record<string, string>,
        )
      : {},
  );

  let columnFormatters = $derived(
    Object.entries(tableConfig.columnFormatters).reduce(
      (acc, [col, formatter]) => {
        if (formatter === "dollars") acc[col] = formatDollars;
        else if (formatter === "percent") acc[col] = formatPercent;
        else acc[col] = formatter as <T>(value: T) => string | T;
        return acc;
      },
      {} as Record<string, <T>(value: T) => string | T>,
    ),
  );

  function indexData(data: Record<string, any>[]) {
    // add an id to each row in the data array. This is useful for identifying
    // the row when the id is not defined in the data. This is a noop function
    // if the id is defined in the data.

    // check the first row to see if it has an id field
    if (data.length > 0 && data[0].id !== undefined) {
      idVariableName = "id";
      return data;
    }

    // if the first row does not have an id field, set the idVariableName to the
    // derived id field, add an id field to each row
    idVariableName = `_idx_${instanceId}`;
    return data.map((row, idx) => ({ ...row, [idVariableName]: idx }));
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
    // generate an instance id for the component to be used to identify the row
    // in the slot. This is useful when the id is not defined in the data.
    instanceId = Math.random().toString(36).substring(7);
    updateInternalTableData();
  });

  $effect(() => {
    if (tableData) {
      updateInternalTableData();
    }
  });
</script>

{#if Array.isArray(internalTableData) && internalTableData.length > 0}
  <!-- <DownloadData data={internalTableData} {columns} dlFileName="report.csv" /> -->
  <div class="flex flex-wrap">
    {#each filters as f (f.key)}
      <div class="mb-2 mr-2 flex rounded-sm border border-blue-700 bg-blue-200 px-2 py-1">
        <Icon icon="mdi:close-circle" class="w-5 pr-1" onclick={() => removeFilter(f)} />
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
                <button
                  class="hover:underline"
                  title="sort"
                  onclick={(event) => {
                    event.preventDefault();
                    sort(col);
                  }}
                >
                  {col}
                </button>
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
            {#if rowActions}
              <th></th>
            {/if}
          </tr>
        </thead>
        <tbody>
          <!-- idVariableName is either 'id' or, if the data doesn't contain an
          id, it's the _idx_ string generated for each row. -->
          {#each internalTableData as row (row[idVariableName])}
            <tr>
              {#each columns as col}
                <td class="pr-4" class:text-right={columnAlignments[col] === "text-right"}>
                  <button class="hover:underline" onclick={() => addFilter(col, row[col])}>
                    {formatCell(col, row[col])}
                  </button>
                </td>
              {/each}
              {#if rowActions}
                <!-- pass the row id to the snippet and also the index of the
                row in the external tableData array. This is useful for
                identifying the row when the id is not defined. To implement
                this, we should write the original tableData index to the
                internalTableData array every time the prop is updated. We
                should ensure that the name of the id doesn't overlap with any
                of the column names in the tableData array by appending a unique
                string to the _idx_ name so that each invocation of the
                component will have a unique id. We will skip over this key when
                rendering the table.
               -->
                <td>
                  {@render rowActions(row[idVariableName])}
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
