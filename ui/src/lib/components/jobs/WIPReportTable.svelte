<script lang="ts">
  import { resolve } from "$app/paths";
  import { formatCurrency } from "$lib/utilities";
  import type { rankWIP } from "$lib/reports/wipReports";
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
  import WIPFactorNotice from "./WIPFactorNotice.svelte";
  import WIPPercentage from "./WIPPercentage.svelte";
  import WIPNoRateSheet from "./WIPNoRateSheet.svelte";
  let {
    rows,
    title,
    includeExpenses,
    includePOs,
  }: {
    rows: ReturnType<typeof rankWIP>;
    title: string;
    includeExpenses: boolean;
    includePOs: boolean;
  } = $props();
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex (Keyboard users need to scroll the table.) -->
<div class="overflow-x-auto" role="region" aria-label={title} tabindex="0">
  <table class="w-full text-sm tabular-nums">
    <caption class="sr-only">{title}</caption>
    <thead class="bg-neutral-100">
      <tr class="border-b border-neutral-300 whitespace-nowrap">
        <th scope="col" class="px-2 py-2 text-left">Project</th>
        <th scope="col" class="px-2 py-2 text-left">Client</th>
        <th scope="col" class="px-2 py-2 text-right">Project value</th>
        <th scope="col" class="px-2 py-2 text-right">Time</th>
        {#if includeExpenses}<th scope="col" class="px-2 py-2 text-right">Expenses</th>{/if}
        {#if includePOs}<th scope="col" class="px-2 py-2 text-right">POs</th>{/if}
        <th scope="col" class="px-2 py-2 text-right">% of project</th>
        <th scope="col" class="px-2 py-2 text-right">Remaining</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as { data, view } (data.id)}
        <tr
          class="h-9 border-b border-neutral-200 hover:bg-neutral-50"
          data-testid={`report-${data.id}`}
        >
          <th scope="row" class="px-2 py-0 text-left font-normal whitespace-nowrap">
            <span class="inline-flex items-center">
              <a
                class="font-semibold text-blue-700 underline"
                href={resolve("/jobs/[jid]/details", { jid: data.id })}>{data.number}</a
              >
              <TimeSummaryHelp
                title={`Project details: ${data.number}`}
                icon="mdi:text-box-outline"
              >
                <p class="font-medium">{data.description}</p>
                <dl class="space-y-1">
                  <div>
                    <dt class="inline font-medium">Client:</dt>
                    <dd class="inline">{data.client || "—"}</dd>
                  </div>
                  <div>
                    <dt class="inline font-medium">Manager:</dt>
                    <dd class="inline">{data.manager || "—"}</dd>
                  </div>
                  <div>
                    <dt class="inline font-medium">Branch:</dt>
                    <dd class="inline">{data.branch || "No branch"}</dd>
                  </div>
                </dl>
              </TimeSummaryHelp>
            </span>
          </th>
          <td class="px-2 py-0"
            ><span class="block max-w-64 truncate" title={data.client}>{data.client || "—"}</span
            ></td
          >
          <td class="px-2 py-0 text-right whitespace-nowrap"
            >{formatCurrency(data.project_value)}</td
          >
          {#each view.rows.filter((row) => row.included) as row (row.key)}
            <td class="px-2 py-0 text-right whitespace-nowrap">
              <span class="inline-flex items-center">
                {#if row.key === "time"}
                  <WIPFactorNotice {data} {row} jobId={data.id} compact />
                  <WIPNoRateSheet {data} compact />
                {/if}
                <span>{row.partial && row.value === 0 ? "—" : formatCurrency(row.value)}</span>
                {#if row.key !== "time"}<WIPFactorNotice
                    {data}
                    {row}
                    jobId={data.id}
                    compact
                  />{/if}
              </span>
            </td>
          {/each}
          <td class="px-2 py-0 text-right whitespace-nowrap">
            <WIPPercentage percent={view.percent} />
          </td>
          <td
            class="px-2 py-0 text-right whitespace-nowrap"
            class:text-red-700={view.balance !== null && view.balance < 0}
            title={view.balance === null
              ? "Unavailable: some included amounts cannot be priced."
              : undefined}
          >
            {view.balance === null ? "—" : formatCurrency(view.balance)}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
