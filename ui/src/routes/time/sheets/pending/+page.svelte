<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { pb } from "$lib/pocketbase";
  import type { PageData } from "./$types";
  import type { TimeSheetTallyQueryRow } from "$lib/utilities";
  import { shortDate } from "$lib/utilities";
  import { globalStore } from "$lib/stores/global";

  let { data }: { data: PageData } = $props();
  let items: TimeSheetTallyQueryRow[] = $state(data.items);

  async function approve(id: string) {
    try {
      await pb.send(`/api/time_sheets/${id}/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      // remove from list after approving
      items = items.filter((ts: TimeSheetTallyQueryRow) => ts.id !== id);
    } catch (error: any) {
      globalStore.addError(error?.response?.message);
    }
  }
</script>

<DsList items={items as TimeSheetTallyQueryRow[]} search={true} inListHeader="Pending Time Sheets">
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
    {#if tally.committed !== ""}
      <span>Committed on {shortDate(tally.committed, true)} by {tally.committer_name}</span>
    {/if}
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
