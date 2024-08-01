<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import type { TimeSheetsRecord } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";

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
    } catch (error) {
      console.error("Error:", error);
    }
  }
</script>

{#snippet anchor({ week_ending })}{week_ending}{/snippet}
{#snippet headline()}<span>placeholder</span>{/snippet}
{#snippet actions({ id })}
  <button onclick={() => unbundle(id)}>unbundle</button>
  <span>recall</span>
  <span>reject</span>
  <span>approve</span>
{/snippet}

<!-- Show the list of items here -->
<DsList
  items={$globalStore.time_sheets as TimeSheetsRecord[]}
  search={true}
  {anchor}
  {headline}
  {actions}
/>
