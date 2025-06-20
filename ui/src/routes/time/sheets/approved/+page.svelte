<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import type { PageData } from "./$types";
  import type { TimeSheetTallyQueryRow } from "$lib/utilities";
  import { shortDate } from "$lib/utilities";

  let { data }: { data: PageData } = $props();
  let items: TimeSheetTallyQueryRow[] = $state(data.items);
</script>

<DsList items={items as TimeSheetTallyQueryRow[]} search={true} inListHeader="Approved Time Sheets">
  {#snippet anchor({ id, week_ending }: TimeSheetTallyQueryRow)}
    <a class="font-bold hover:underline" href={`/time/sheets/${id}/details`}
      >{shortDate(week_ending, true)}</a
    >
  {/snippet}
  {#snippet headline(tally: TimeSheetTallyQueryRow)}
    {tally.given_name} {tally.surname}
  {/snippet}
  {#snippet byline(tally: TimeSheetTallyQueryRow)}
    {#if tally.work_total_hours > 0}
      <span>{tally.work_total_hours} hours worked</span>
    {:else}
      <span>no work</span>
    {/if}
  {/snippet}
  {#snippet line1(tally: TimeSheetTallyQueryRow)}
    <span>{tally.non_work_total_hours} hours off</span>
  {/snippet}
  {#snippet line2(tally: TimeSheetTallyQueryRow)}
    {#if tally.approved !== ""}
      <div class="flex items-center gap-1">
        <DsLabel color="green">Approved</DsLabel>
        <span>on {shortDate(tally.approved, true)}</span>
      </div>
    {/if}
  {/snippet}
  {#snippet line3(tally: TimeSheetTallyQueryRow)}
    {#if tally.committed !== ""}
      <div class="mt-1 flex items-center gap-1">
        <DsLabel color="blue">Committed</DsLabel>
        <span>on {shortDate(tally.committed, true)} by {tally.committer_name}</span>
      </div>
    {/if}
  {/snippet}
</DsList>
