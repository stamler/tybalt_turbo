<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";

  interface JobPOEntry {
    id: string;
    po_number: string;
    date: string;
    total: number;
    type: string;
    division_code: string;
    surname: string;
    given_name: string;
  }

  interface $$Props {
    summary: Record<string, any>;
    items: JobPOEntry[];
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
      Purchase orders referencing a job belong to the branch of the job they reference.
    </span>
    <div><span class="font-semibold">Total:</span> {summary.total_amount ?? 0}</div>
    {#if summary.earliest_po}
      <div>
        <span class="font-semibold">Date Range:</span>
        {summary.earliest_po} – {summary.latest_po}
      </div>
    {/if}
  </div>
</div>

{#if listLoading && items.length === 0}
  <div class="px-4">Loading…</div>
{:else if items.length === 0}
  <div class="px-4">No <i>Active</i> purchase orders found.</div>
{:else}
  <div class="w-full overflow-hidden">
    <DsList {items} search={false} inListHeader="Purchase Orders">
      {#snippet anchor(item: JobPOEntry)}{item.date}{/snippet}
      {#snippet headline(item: JobPOEntry)}{item.po_number}{/snippet}
      {#snippet byline(item: JobPOEntry)}{item.given_name} {item.surname}{/snippet}
      {#snippet line1(item: JobPOEntry)}
        <span class="font-bold">{item.division_code}</span>
        <span class="font-bold">{item.type}</span>
        {item.po_number}
      {/snippet}
    </DsList>
    {#if page < totalPages}
      <div class="mt-4 text-center">
        <button
          class="rounded bg-blue-600 px-4 py-2 text-white"
          onclick={loadMore}
          disabled={listLoading}
        >
          {listLoading ? "Loading…" : "Load More"}
        </button>
      </div>
    {/if}
  </div>
{/if}
