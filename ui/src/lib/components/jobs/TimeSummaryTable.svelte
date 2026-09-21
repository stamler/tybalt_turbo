<script lang="ts">
  import DSActionButton from "$lib/components/DSActionButton.svelte";
  import { downloadCsvRows } from "$lib/utilities";
  import TimeValuationNote from "./TimeValuationNote.svelte";
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
  import TimeSummaryValue from "./TimeSummaryValue.svelte";
  import {
    summaryCell,
    summaryName,
    summaryTotals,
    visibleSummaryRows,
    type RateSheet,
    type SummaryColumn,
    type SummaryFilter,
    type SummaryKind,
    type TimeSummaryRow,
  } from "./timeSummary";

  let {
    rows,
    kind,
    rateSheet,
    filename,
  }: {
    rows: TimeSummaryRow[];
    kind: SummaryKind;
    rateSheet?: RateSheet;
    filename: string;
  } = $props();
  let filters = $state<SummaryFilter[]>([]);
  let sortColumn = $state<SummaryColumn>("name");
  let sortDirection = $state<0 | 1 | -1>(0);
  const columns: SummaryColumn[] = ["name", "hours", "value", "percent"];
  const labels = $derived({
    name: kind === "staff" ? "Staff member" : "Division",
    hours: "Hours",
    value: "Value",
    percent: "Share",
  });
  const visibleRows = $derived(visibleSummaryRows(rows, kind, filters, sortColumn, sortDirection));
  const totals = $derived(summaryTotals(visibleRows));
  const allTotals = $derived(summaryTotals(rows));
  const noRateSheet = $derived(
    rows.length > 0 ? rows[0].rate_sheet_revision === null : !rateSheet?.id,
  );
  const totalLabel = $derived(
    `${filters.length ? "Visible " : ""}${totals.unpriced_hours > 0 ? "known total" : "total"}`,
  );

  function sort(column: SummaryColumn) {
    sortDirection =
      column !== sortColumn ? 1 : sortDirection === 0 ? 1 : sortDirection === 1 ? -1 : 0;
    sortColumn = column;
  }

  function filterBy(row: TimeSummaryRow, column: SummaryColumn) {
    filters = [
      ...filters.filter((f) => f.column !== column),
      { column, value: summaryCell(row, column, kind) },
    ];
  }
</script>

<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
  <div>
    {#if noRateSheet}
      <TimeSummaryHelp
        label={allTotals.estimated_hours > 0
          ? "No rate sheet · employee rates used"
          : "No rate sheet"}
        title="No rate sheet assigned"
        warning={true}
      >
        <p>
          This job has no rate sheet. Employee default charge-out rates are used when they are
          greater than zero.
        </p>
        {#if allTotals.estimated_hours > 0}<p>
            {allTotals.estimated_hours.toFixed(2)} hours use employee defaults. Their estimated value
            is included in the total.
          </p>{/if}
        {#if allTotals.unpriced_hours > 0}<p>
            {allTotals.unpriced_hours.toFixed(2)} hours cannot be priced. Affected amounts are marked
            Partial.
          </p>{/if}
      </TimeSummaryHelp>
    {/if}
  </div>
  <TimeValuationNote {rateSheet} />
</div>

{#if filters.length}
  <div class="mb-2 flex flex-wrap gap-2" aria-label="Table filters">
    {#each filters as filter (filter.column)}
      <button
        type="button"
        class="rounded-sm bg-blue-100 px-2 py-1 text-sm text-blue-900"
        onclick={() => (filters = filters.filter((f) => f.column !== filter.column))}
        aria-label={`Remove ${labels[filter.column]} filter`}
      >
        {labels[filter.column]}: {filter.value ?? "—"} ×
      </button>
    {/each}
  </div>
{/if}

{#if rows.length === 0}
  <p class="py-3 text-neutral-600">No time entries found for this date range.</p>
{:else}
  <div class="overflow-x-auto">
    <table class="w-full text-sm tabular-nums">
      <caption class="sr-only">{kind === "staff" ? "Staff" : "Divisions"} time summary</caption>
      <thead>
        <tr>
          {#each columns as column (column)}
            <th
              scope="col"
              class="border-b border-neutral-300 px-2 py-2 font-semibold"
              class:text-left={column === "name"}
              class:text-right={column !== "name"}
              aria-sort={sortColumn !== column || sortDirection === 0
                ? "none"
                : sortDirection === 1
                  ? "ascending"
                  : "descending"}
            >
              <button
                type="button"
                class="whitespace-nowrap hover:underline"
                onclick={() => sort(column)}
              >
                {labels[column]}{sortColumn === column && sortDirection
                  ? sortDirection === 1
                    ? " ↑"
                    : " ↓"
                  : ""}
              </button>
            </th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each visibleRows as row (row)}
          {@const name = summaryName(row, kind)}
          <tr>
            <th scope="row" class="border-b border-neutral-200 px-2 py-3 text-left font-normal">
              <button
                type="button"
                class="text-left hover:underline"
                onclick={() => filterBy(row, "name")}>{name}</button
              >
            </th>
            <td class="border-b border-neutral-200 px-2 py-3 text-right">
              <button type="button" class="hover:underline" onclick={() => filterBy(row, "hours")}
                >{row.hours.toFixed(2)}</button
              >
            </td>
            <td class="border-b border-neutral-200 px-2 py-3 text-right">
              <TimeSummaryValue
                value={row.value}
                hours={row.hours}
                estimatedHours={row.estimated_hours}
                unpricedHours={row.unpriced_hours}
                {noRateSheet}
                context={name}
                onFilter={() => filterBy(row, "value")}
              />
            </td>
            <td class="border-b border-neutral-200 px-2 py-3 text-right">
              <button type="button" class="hover:underline" onclick={() => filterBy(row, "percent")}
                >{row.percent === null ? "—" : `${row.percent.toFixed(1)}%`}</button
              >
            </td>
          </tr>
        {:else}
          <tr><td colspan="4" class="px-2 py-3 text-neutral-600">No matching rows.</td></tr>
        {/each}
      </tbody>
      {#if visibleRows.length}
        <tfoot>
          <tr>
            <th scope="row" class="px-2 py-3 text-left font-semibold"
              >{totalLabel[0].toUpperCase() + totalLabel.slice(1)}</th
            >
            <td class="px-2 py-3 text-right font-semibold">{totals.hours.toFixed(2)}</td>
            <td class="px-2 py-3 text-right font-semibold">
              <TimeSummaryValue
                value={totals.value}
                hours={totals.hours}
                estimatedHours={totals.estimated_hours}
                unpricedHours={totals.unpriced_hours}
                {noRateSheet}
                context={totalLabel}
              />
            </td>
            <td class="px-2 py-3 text-right font-semibold"
              >{totals.percent === null ? "—" : `${totals.percent.toFixed(1)}%`}</td
            >
          </tr>
        </tfoot>
      {/if}
    </table>
  </div>
  <div class="mt-3 flex items-center justify-between gap-3">
    <span class="text-sm text-neutral-500">CAD</span>
    <DSActionButton
      title="Download full summary CSV"
      icon="mdi:download"
      color="yellow"
      action={() => downloadCsvRows(filename, rows)}
    />
  </div>
{/if}
