<script lang="ts">
  import WIPTimeDetails from "./WIPTimeDetails.svelte";
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
  import type { JobWIP, wipView } from "./wip";
  let {
    data,
    row,
    compact = false,
    jobId = "",
  }: {
    compact?: boolean;
    jobId?: string;
    data: JobWIP;
    row: ReturnType<typeof wipView>["rows"][number];
  } = $props();
</script>

<!-- NR covers employee-rate estimates when no sheet is assigned. -->
{#if row.partial || (row.estimated && !(compact && row.key === "time" && data.no_rate_sheet))}
  <TimeSummaryHelp
    label={row.partial ? "Partial" : "Estimated"}
    title={`${row.name}: value details`}
    warning
    icon="mdi:alert-outline"
    symbol={row.partial ? undefined : "≈"}
    iconOnly={compact}
  >
    {#if row.key === "time"}
      <WIPTimeDetails {data} {jobId} />
    {:else if row.key === "expenses"}
      <p>
        {data.unpriced_expenses} committed expenses have no settled CAD amount. Their values are excluded.
      </p>
    {:else}
      {#if data.estimated_pos > 0}
        <p>{data.estimated_pos} active PO balances use current exchange rates.</p>
      {/if}
      {#if data.unpriced_pos > 0}
        <p>
          {data.unpriced_pos} active POs cannot be valued because an exchange rate or recurring approval
          value is missing. Their values are excluded.
        </p>
      {/if}
    {/if}
  </TimeSummaryHelp>
{/if}
