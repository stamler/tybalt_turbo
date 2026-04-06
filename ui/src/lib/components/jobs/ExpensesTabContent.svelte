<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import {
    currencyIconHref,
    formatCurrencyAmount,
    normalizeCurrencyCode,
  } from "$lib/utilities";

  interface JobExpenseEntry {
    id: string;
    date: string;
    total: number;
    currency: string;
    currency_code: string;
    currency_symbol: string;
    currency_icon: string;
    description: string;
    committed_week_ending: string;
    division_code: string;
    payment_type: string;
    surname: string;
    given_name: string;
    category_name: string;
  }

  interface $$Props {
    summary: Record<string, any>;
    items: JobExpenseEntry[];
    listLoading: boolean;
    loadMore: () => void;
    page: number;
    totalPages: number;
  }

  let { summary, items, listLoading, loadMore, page, totalPages } = $props();
</script>

<div class="px-4">
  <div class="space-y-1">
    <span class="text-sm text-gray-500">
      Expenses referencing a job belong to the branch of the job they reference.
    </span>
    <div>
      <span class="font-semibold">Total Settled (CAD):</span>
      {formatCurrencyAmount(summary.total_amount ?? 0, "CAD")}
    </div>
    {#if summary.earliest_expense}
      <div>
        <span class="font-semibold">Date Range:</span>
        {summary.earliest_expense} – {summary.latest_expense}
      </div>
    {/if}
  </div>
</div>

{#if listLoading && items.length === 0}
  <div class="px-4">Loading…</div>
{:else if items.length === 0}
  <div class="px-4">No <i>Committed</i> expenses found.</div>
{:else}
  <div class="w-full overflow-hidden">
    <DsList {items} search={false} inListHeader="Expenses">
      {#snippet anchor(item: JobExpenseEntry)}<a
          href={`/expenses/${item.id}/details`}
          class="text-blue-600 hover:underline">{item.date}</a
        >{/snippet}
      {#snippet headline(item: JobExpenseEntry)}
        <span class="inline-flex items-center gap-2">
          {#if normalizeCurrencyCode(item.currency_code) !== "CAD" && item.currency_icon}
            <img
              src={currencyIconHref(item.currency, item.currency_icon)}
              alt={`${item.currency_code} icon`}
              class="h-5 w-5 rounded-full border border-neutral-200 object-cover"
            />
          {/if}
          <span>{formatCurrencyAmount(item.total, item.currency_code)}</span>
        </span>
      {/snippet}
      {#snippet byline(item: JobExpenseEntry)}{item.given_name} {item.surname}{/snippet}
      {#snippet line1(item: JobExpenseEntry)}
        <span class="font-bold">{item.division_code}</span>
        <span class="font-bold">{item.payment_type}</span>
        {item.description}
      {/snippet}
    </DsList>
    {#if page < totalPages}
      <div class="mt-4 text-center">
        <button
          class="rounded-sm bg-blue-600 px-4 py-2 text-white"
          onclick={loadMore}
          disabled={listLoading}
        >
          {listLoading ? "Loading…" : "Load More"}
        </button>
      </div>
    {/if}
  </div>
{/if}
