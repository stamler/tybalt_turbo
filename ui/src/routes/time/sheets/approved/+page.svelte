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
    {#if tally.work_total_hours > 0}
      <span>{tally.work_total_hours} hours worked</span>
    {:else}
      <span>no work</span>
    {/if}
  {/snippet}
  {#snippet byline(tally: TimeSheetTallyQueryRow)}
    <span>{tally.non_work_total_hours} hours off</span>
  {/snippet}
  {#snippet line3(tally: TimeSheetTallyQueryRow)}
    <DsLabel color="green">Approved</DsLabel>
  {/snippet}
</DsList>
