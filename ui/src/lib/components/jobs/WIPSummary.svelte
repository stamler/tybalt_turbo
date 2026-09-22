<script lang="ts">
  import { formatCurrency } from "$lib/utilities";
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
  import { wipView, type JobWIP } from "./wip";
  let { data }: { data: JobWIP } = $props();
  let includeExpenses = $state(true);
  let includePOs = $state(true);
  let showChart = $state(false);
  const view = $derived(wipView(data, includeExpenses, includePOs));
  const chartId = $props.id();
  const percent = (value: number | null) => (value === null ? "—" : `${value.toFixed(1)}%`);
</script>

<section class="max-w-4xl space-y-5" aria-label="Work in progress">
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div>
      <h2 class="text-lg font-semibold">Work in progress</h2>
      <p class="text-sm text-neutral-600">Job to date · {data.as_of} · CAD</p>
    </div>
    <TimeSummaryHelp title="How WIP is calculated">
      <p>
        WIP compares recorded time, included expenses, and remaining active PO commitments with the
        project value. It does not measure work completed or cash paid.
      </p>
      <p>
        Time uses the same job rates and employee defaults as the Staff and Divisions summaries,
        including committed and uncommitted entries. Meal hours, amendments, and overtime billing
        are excluded.
      </p>
      <p>
        Expenses include committed, non-rejected records through the report date. Foreign expenses
        use settled CAD amounts.
      </p>
      <p>
        Active POs include future commitments. Their remaining value subtracts committed expenses,
        even when Include committed expenses is off. Recurring POs use the full approved value
        across all occurrences. Each remaining balance is at least zero.
      </p>
      <p>
        Foreign PO balances are converted using the current exchange rate, so they are estimates.
        Amounts with missing settlement, exchange rates, or recurring approval values are marked
        Partial.
      </p>
      <p>
        The table and budget bar show percentages of project value. The optional chart shows shares
        of the included total. Only this job is included; child jobs are separate.
      </p>
    </TimeSummaryHelp>
  </div>

  <div class="flex flex-wrap gap-x-6 gap-y-3">
    <label class="flex items-center gap-2"
      ><input type="checkbox" bind:checked={includeExpenses} />Include committed expenses</label
    >
    <label class="flex items-center gap-2"
      ><input type="checkbox" bind:checked={includePOs} />Include active POs</label
    >
  </div>

  {#if data.no_rate_sheet}
    <TimeSummaryHelp
      label="No rate sheet · employee rates used where available"
      title="Time uses employee defaults"
      warning
    >
      <p>
        This job has no rate sheet. Time uses employee default charge-out rates greater than zero. {data.estimated_hours.toFixed(
          2,
        )} hours use defaults; {data.unpriced_hours.toFixed(2)} hours remain unpriced.
      </p>
    </TimeSummaryHelp>
  {/if}

  <div
    class="rounded-lg border border-neutral-200 bg-neutral-50 p-4"
    aria-live="polite"
    aria-atomic="true"
  >
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="text-sm text-neutral-600">
          {view.partial ? "Known included value" : "Included value"}
        </p>
        <p class="text-2xl font-semibold tabular-nums" data-testid="wip-total">
          {view.partial && view.total === 0 ? "—" : formatCurrency(view.total)}
        </p>
        <p class="mt-1 text-sm text-neutral-600">
          Project value: {formatCurrency(data.project_value)}
        </p>
      </div>
      <div class="text-right">
        <p class="text-2xl font-semibold tabular-nums" data-testid="wip-percent">
          {percent(view.percent)}
        </p>
        <p class="text-sm text-neutral-600">of project value used or committed</p>
      </div>
    </div>
    {#if view.partial}
      <p class="mt-3 text-sm text-amber-800">
        Partial — some included amounts cannot be valued. The total excludes those amounts; the
        overall percentage and charts are unavailable.
      </p>
    {:else if view.estimated}
      <p class="mt-3 text-sm text-amber-800">
        Includes estimates. Open the notices below for details.
      </p>
    {/if}
    {#if data.project_value <= 0}
      <p class="mt-3 text-sm text-neutral-600">
        Set a project value greater than zero to see budget percentages.
      </p>
    {:else if view.balance !== null}
      <p class="mt-3 text-sm" class:text-red-700={view.balance < 0} data-testid="wip-balance">
        {formatCurrency(Math.abs(view.balance))}
        {view.balance < 0 ? "over project value" : "of project value remaining"}
      </p>
    {/if}
  </div>

  {#if view.chartable && data.project_value > 0}
    <figure aria-label={`Budget use: ${percent(view.percent)} of project value`}>
      <figcaption class="mb-2 text-sm font-medium">Budget use · % of project value</figcaption>
      <div class="relative" data-testid="wip-budget-bar">
        <div class="flex h-5 overflow-hidden rounded-sm bg-neutral-200">
          {#each view.rows.filter((row) => row.included) as row (row.key)}
            <div
              style:width={`${row.width}%`}
              style:background-color={row.color}
              title={`${row.name}: ${percent(row.percent)}`}
            >
              <span class="sr-only">{row.name}: {percent(row.percent)}.</span>
            </div>
          {/each}
        </div>
        <div
          class="absolute -top-1 h-7 border-l-2 border-neutral-900"
          style:left={`calc(${view.budgetMarker}% - 2px)`}
          aria-hidden="true"
        ></div>
      </div>
      <p class="mt-2 text-sm text-neutral-600">
        Black marker: 100% of project value. Grey: unused value.
      </p>
    </figure>
  {/if}

  <div class="overflow-x-auto">
    <table class="w-full text-sm tabular-nums">
      <caption class="sr-only">WIP components and percentages of project value</caption>
      <thead
        ><tr class="border-b border-neutral-300"
          ><th scope="col" class="px-2 py-2 text-left">Component</th><th
            scope="col"
            class="px-2 py-2 text-right">Value</th
          ><th scope="col" class="px-2 py-2 text-right">% of project</th></tr
        ></thead
      >
      <tbody>
        {#each view.rows as row (row.key)}
          <tr
            class="border-b border-neutral-200"
            class:text-neutral-500={!row.included}
            data-testid={`wip-${row.key}`}
          >
            <th scope="row" class="px-2 py-3 text-left font-normal">
              <span
                class="mr-2 inline-block h-2.5 w-2.5 rounded-full"
                style:background-color={row.color}
                aria-hidden="true"
              ></span>{row.name}
              {#if row.key === "time"}
                <TimeSummaryHelp title="How time value is calculated">
                  <p>Hours × job rate, with employee defaults where needed.</p>
                </TimeSummaryHelp>
              {/if}
              {#if !row.included}<span class="ml-2 text-xs">Excluded</span>{/if}
            </th>
            <td class="px-2 py-3 text-right">
              <div>{row.partial && row.value === 0 ? "—" : formatCurrency(row.value)}</div>
              {#if row.partial || row.estimated}
                <TimeSummaryHelp
                  label={row.partial ? "Partial" : "Estimated"}
                  title={`${row.name}: value details`}
                  warning
                >
                  {#if row.key === "time"}
                    <p>
                      {data.hours.toFixed(2)} recorded hours. {data.estimated_hours.toFixed(2)} hours
                      use employee default charge-out rates; {data.unpriced_hours.toFixed(2)} hours cannot
                      be priced and are excluded from the value.
                    </p>
                  {:else if row.key === "expenses"}
                    <p>
                      {data.unpriced_expenses} committed expenses have no settled CAD amount. Their values
                      are excluded.
                    </p>
                  {:else}
                    <p>
                      {data.estimated_pos} active PO balances use current exchange rates. {data.unpriced_pos}
                      active POs cannot be valued because an exchange rate or recurring approval value
                      is missing. Their values are excluded.
                    </p>
                  {/if}
                </TimeSummaryHelp>
              {/if}
            </td>
            <td class="px-2 py-3 text-right">{percent(row.percent)}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  <button
    type="button"
    class="rounded-sm border border-neutral-300 px-3 py-2 text-sm hover:bg-neutral-50"
    aria-expanded={showChart}
    aria-controls={chartId}
    onclick={() => (showChart = !showChart)}
  >
    {showChart ? "Hide breakdown chart" : "Show breakdown chart"}
  </button>
  {#if showChart}
    <div id={chartId}>
      {#if view.chartable && view.total > 0}
        <figure class="flex flex-wrap items-center gap-6" aria-label="Share of included total">
          <svg
            viewBox="0 0 120 120"
            class="h-48 w-48 shrink-0"
            role="img"
            aria-label="Doughnut chart of included WIP components"
          >
            {#each view.rows.filter((row) => row.included && row.share > 0) as row (row.key)}
              <circle
                cx="60"
                cy="60"
                r="45"
                fill="none"
                stroke={row.color}
                stroke-width="16"
                pathLength="100"
                stroke-dasharray={`${row.share} ${100 - row.share}`}
                stroke-dashoffset={-row.offset}
                transform="rotate(-90 60 60)"
                ><title>{row.name}: {row.share.toFixed(1)}% of included total</title></circle
              >
            {/each}
          </svg>
          <figcaption>
            <p class="mb-2 font-medium">Share of included total</p>
            {#each view.rows.filter((row) => row.included) as row (row.key)}
              <p class="py-1 text-sm">
                <span
                  class="mr-2 inline-block h-2.5 w-2.5 rounded-full"
                  style:background-color={row.color}
                  aria-hidden="true"
                ></span>{row.name}: <strong>{row.share.toFixed(1)}%</strong>
              </p>
            {/each}
            <p class="mt-2 text-sm text-neutral-600">
              Based on {formatCurrency(view.total)}, not project value.
            </p>
          </figcaption>
        </figure>
      {:else}
        <p class="text-sm text-neutral-600">
          The chart needs a complete, positive total with no negative components.
        </p>
      {/if}
    </div>
  {/if}
</section>
