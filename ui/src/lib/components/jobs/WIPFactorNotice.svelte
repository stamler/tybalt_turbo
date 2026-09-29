<script lang="ts">
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
  import type { JobWIP, wipView } from "./wip";
  let { data, row }: { data: JobWIP; row: ReturnType<typeof wipView>["rows"][number] } = $props();
</script>

{#if row.partial || row.estimated}
  <TimeSummaryHelp
    label={row.partial ? "Partial" : "Estimated"}
    title={`${row.name}: value details`}
    warning
  >
    {#if row.key === "time"}
      <p>
        {data.hours.toFixed(2)} recorded hours. {data.estimated_hours.toFixed(2)} hours use employee default
        charge-out rates; {data.unpriced_hours.toFixed(2)} hours cannot be priced and are excluded from
        the value.
      </p>
    {:else if row.key === "expenses"}
      <p>
        {data.unpriced_expenses} committed expenses have no settled CAD amount. Their values are excluded.
      </p>
    {:else}
      <p>
        {data.estimated_pos} active PO balances use current exchange rates. {data.unpriced_pos}
        active POs cannot be valued because an exchange rate or recurring approval value is missing. Their
        values are excluded.
      </p>
    {/if}
  </TimeSummaryHelp>
{/if}
