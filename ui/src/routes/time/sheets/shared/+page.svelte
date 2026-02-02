<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { shortDate } from "$lib/utilities";
  import type { TimeSheetTallyQueryRow } from "$lib/utilities";
  import type { PageData } from "./$types";
  import { untrack } from "svelte";

  // `data` is injected by the load function in +page.ts
  let { data }: { data: PageData } = $props();
  let items: TimeSheetTallyQueryRow[] = $state(untrack(() => data.items));
</script>

<DsList items={items as TimeSheetTallyQueryRow[]} search={true} inListHeader="Shared Time Sheets">
  {#snippet anchor({ id, week_ending }: TimeSheetTallyQueryRow)}
    <a class="font-bold hover:underline" href={`/time/sheets/${id}/details`}
      >{shortDate(week_ending, true)}</a
    >
  {/snippet}
  {#snippet headline({ given_name, surname }: TimeSheetTallyQueryRow)}
    {given_name} {surname}
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
  {#snippet actions({ id }: TimeSheetTallyQueryRow)}
    <DsActionButton
      action={`/time/sheets/${id}/details`}
      icon="mdi:details"
      title="Details"
      color="purple"
    />
  {/snippet}
</DsList>
