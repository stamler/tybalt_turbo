<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import type { TimeEntriesResponse, TimeSheetsResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";
  import { goto } from "$app/navigation";
  import { calculateTallies, shortDate } from "$lib/utilities";

  let errors = $state({} as any);

  async function unbundle(timeSheetId: string) {
    try {
      const response = await pb.send("/api/unbundle-timesheet", {
        method: "POST",
        body: JSON.stringify({ timeSheetId }),
        headers: {
          "Content-Type": "application/json",
        },
      });

      // refresh the time sheets list in the global store
      globalStore.refresh("time_sheets");

      // navigate to the time entries list to show the unbundled time entries
      goto(`/time/entries/list`);
    } catch (error) {
      console.error("Error:", error);
    }
  }
</script>

{#snippet anchor({ week_ending }: TimeSheetsResponse)}
  <a href="/time/sheets/details">{shortDate(week_ending)}</a>
{/snippet}
{#snippet headline()}<span>placeholder</span>{/snippet}
{#snippet line1({ expand }: TimeSheetsResponse)}
  <span>
    {JSON.stringify(calculateTallies((expand as { "time_entries(tsid)": TimeEntriesResponse[]})["time_entries(tsid)"]))}
  </span>
{/snippet}
{#snippet actions({ id }: TimeSheetsResponse)}
  <button onclick={() => unbundle(id)}>unbundle</button>
  <span>recall</span>
  <span>reject</span>
  <span>approve</span>
{/snippet}

<!-- Show the list of items here -->
<DsList
  items={$globalStore.time_sheets as TimeSheetsResponse[]}
  search={true}
  {anchor}
  {headline}
  {line1}
  {actions}
/>
