<script lang="ts">
  import { resolve } from "$app/paths";
  import { formatCurrency } from "$lib/utilities";
  import type { rankWIP } from "$lib/reports/wipReports";
  import WIPFactorNotice from "./WIPFactorNotice.svelte";
  import WIPNoRateSheet from "./WIPNoRateSheet.svelte";
  let { rows, title }: { rows: ReturnType<typeof rankWIP>; title: string } = $props();
</script>

<div class="overflow-x-auto rounded-lg border border-neutral-200">
  <table class="w-full text-sm tabular-nums">
    <caption class="sr-only">{title}</caption>
    <thead class="bg-neutral-50">
      <tr class="border-b border-neutral-200">
        <th scope="col" class="px-3 py-3 text-left">Project</th>
        <th scope="col" class="px-3 py-3 text-right">Project value</th>
        <th scope="col" class="px-3 py-3 text-right">Time value</th>
        <th scope="col" class="px-3 py-3 text-right">Expenses</th>
        <th scope="col" class="px-3 py-3 text-right">Remaining active POs</th>
        <th scope="col" class="px-3 py-3 text-right">Included value</th>
        <th scope="col" class="px-3 py-3 text-right">% of project</th>
      </tr>
    </thead>
    <tbody>
      {#each rows as { data, view } (data.id)}
        <tr
          class="border-b border-neutral-200 align-top last:border-0"
          data-testid={`report-${data.id}`}
        >
          <th scope="row" class="min-w-64 px-3 py-3 text-left font-normal">
            <a
              class="font-semibold text-blue-700 underline"
              href={resolve("/jobs/[jid]/details", { jid: data.id })}>{data.number}</a
            >
            <p>{data.description}</p>
            <p class="mt-1 text-xs text-neutral-600">
              {data.client} · {data.manager} · {data.branch || "No branch"}
            </p>
            <WIPNoRateSheet {data} />
          </th>
          <td class="px-3 py-3 text-right whitespace-nowrap"
            >{formatCurrency(data.project_value)}</td
          >
          {#each view.rows as row (row.key)}
            <td
              class="px-3 py-3 text-right whitespace-nowrap"
              class:text-neutral-500={!row.included}
            >
              <div>{row.partial && row.value === 0 ? "—" : formatCurrency(row.value)}</div>
              {#if !row.included}<div class="text-xs">Excluded</div>{/if}
              <WIPFactorNotice {data} {row} />
            </td>
          {/each}
          <td class="px-3 py-3 text-right whitespace-nowrap">
            {view.partial && view.total === 0 ? "—" : formatCurrency(view.total)}
            {#if view.partial}<div class="text-xs text-amber-800">Known value only</div>
            {:else if view.estimated}<div class="text-xs text-amber-800">Estimated</div>{/if}
          </td>
          <td
            class="px-3 py-3 text-right font-semibold whitespace-nowrap"
            class:text-red-700={view.percent !== null && view.percent > 100}
          >
            {view.percent === null ? "—" : `${view.percent.toFixed(1)}%`}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
