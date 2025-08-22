<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import { pb } from "$lib/pocketbase";
  import { downloadCSV } from "$lib/utilities";

  interface JobTimeEntry {
    id: string;
    date: string;
    hours: number;
    description: string;
    work_record: string;
    week_ending: string;
    tsid: string;
    division_code: string;
    time_type_code: string;
    surname: string;
    given_name: string;
    category_name: string;
  }

  interface $$Props {
    summary: Record<string, any>;
    items: JobTimeEntry[];
    listLoading: boolean;
    loadMore: () => void;
    page: number;
    totalPages: number;
    jobId: string;
  }

  const { summary, items, listLoading, loadMore, page, totalPages, jobId } = $props();

  async function fetchFullReport() {
    const url = `${pb.baseUrl}/api/jobs/${jobId}/time/full_report`;
    const fileName = `job_full_time_report_${jobId}.csv`;
    await downloadCSV(url, fileName);
  }
</script>

<div class="px-4">
  <div class="space-y-1">
    <div><span class="font-semibold">Total Hours:</span> {summary.total_hours ?? 0}</div>
    {#if summary.earliest_entry}
      <div>
        <span class="font-semibold">Date Range:</span>
        {summary.earliest_entry} – {summary.latest_entry}
      </div>
    {/if}
  </div>
</div>

<div class="mt-2 px-4">
  <button type="button" onclick={fetchFullReport} class="text-blue-600 hover:underline">
    Full Report (CSV)
  </button>
</div>

{#if listLoading && items.length === 0}
  <div class="px-4">Loading…</div>
{:else if items.length === 0}
  <div class="px-4">No entries found.</div>
{:else}
  <div class="w-full overflow-hidden">
    <DsList {items} search={false} inListHeader="Time Entries">
      {#snippet anchor(item: JobTimeEntry)}{item.date}{/snippet}
      {#snippet headline(item: JobTimeEntry)}{item.hours}{/snippet}
      {#snippet byline(item: JobTimeEntry)}{item.given_name} {item.surname}{/snippet}
      {#snippet line1(item: JobTimeEntry)}
        <span class="font-bold">{item.division_code}</span>
        <span class="font-bold">{item.time_type_code}</span>
        {item.description}
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
