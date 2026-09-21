<script lang="ts">
  import { formatCurrency } from "$lib/utilities";
  import TimeSummaryHelp from "./TimeSummaryHelp.svelte";
  import { displayValue, pricingAnomaly } from "./timeSummary";

  let {
    value,
    hours,
    estimatedHours,
    unpricedHours,
    noRateSheet,
    context,
    onFilter,
  }: {
    value: number;
    hours: number;
    estimatedHours: number;
    unpricedHours: number;
    noRateSheet: boolean;
    context: string;
    onFilter?: () => void;
  } = $props();
  const amount = $derived(displayValue({ value, hours, unpriced_hours: unpricedHours }));
  const anomaly = $derived(pricingAnomaly(estimatedHours, unpricedHours, noRateSheet));
  const title = $derived(
    anomaly === "Partial" ? `${context}: unpriced hours` : `${context}: employee rates used`,
  );
</script>

<div class="flex flex-wrap items-center justify-end gap-x-1">
  {#if onFilter}
    <button
      type="button"
      onclick={onFilter}
      aria-label={`Filter by value ${amount === null ? "unavailable" : formatCurrency(amount)}`}
      class="whitespace-nowrap hover:underline"
    >
      {amount === null ? "—" : formatCurrency(amount)}
    </button>
  {:else}
    <span class="whitespace-nowrap">{amount === null ? "—" : formatCurrency(amount)}</span>
  {/if}
  {#if anomaly}
    <TimeSummaryHelp label={anomaly} {title} warning={true}>
      {#if unpricedHours > 0}
        <p>
          {unpricedHours.toFixed(2)} of {hours.toFixed(2)} hours have no matching job rate and no employee
          default rate greater than zero.
        </p>
        <p>
          These hours remain in the hours total but are excluded from the value. Shares are
          unavailable while the job total is incomplete.
        </p>
      {/if}
      {#if estimatedHours > 0}
        <p>
          {estimatedHours.toFixed(2)} of {hours.toFixed(2)} hours use employee default charge-out rates
          because no job rate matches. Their estimated value is included.
        </p>
      {/if}
    </TimeSummaryHelp>
  {/if}
</div>
