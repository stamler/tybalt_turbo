<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import { globalStore } from "$lib/stores/global";
  import { goto } from "$app/navigation";
  import { shortDate, hoursWorked, hoursOff, jobs, divisions } from "$lib/utilities";
  import type { TimeSheetTally } from "$lib/utilities";

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

{#snippet anchor({ week_ending }: TimeSheetTally)}
  <a class="font-bold hover:underline" href="/time/sheets/details">{shortDate(week_ending)}</a>
{/snippet}
{#snippet headline(tally: TimeSheetTally)}
  <span>{hoursWorked(tally)}</span>
{/snippet}
{#snippet byline(tally: TimeSheetTally)}
  <span>/ {hoursOff(tally)}</span>
  {#if tally.offRotationDates.length > 0}
    <span>/ {tally.offRotationDates.length}day(s) off rotation</span>
  {/if}
  {#if tally.bankEntries.length > 0}
    <span>/ {tally.bankEntries.reduce((sum, entry) => sum + entry.hours, 0)} hours banked</span>
  {/if}
{/snippet}
{#snippet line2(tally: TimeSheetTally)}
  <span>{divisions(tally)}</span>
{/snippet}
{#snippet line1(tally: TimeSheetTally)}
  <span>{jobs(tally)}</span>
{/snippet}
{#snippet line3(tally: TimeSheetTally)}
  <!-- TODO: implement rejection message, viewers, reviewed, and payout requests -->
  <span>RejectionStatus, Viewers, Reviewed, PayoutRequests</span>
{/snippet}
{#snippet actions({ id }: TimeSheetTally)}
  <button onclick={() => unbundle(id)}>unbundle</button>
  <span>recall</span>
  <span>reject</span>
  <span>approve</span>
{/snippet}

<!-- Show the list of items here -->
<DsList
  items={$globalStore.time_sheets_tallies}
  search={true}
  {anchor}
  {headline}
  {byline}
  {line1}
  {line2}
  {line3}
  {actions}
/>

<!--

    <template #line3="item">
      <span v-if="item.rejected" style="color: red">
        Rejected: {{ item.rejectionReason }}
      </span>
      <span v-if="Object.keys(unreviewed(item)).length > 0">
        Viewers:
        <span
          class="label"
          v-for="(value, uid) in unreviewed(item)"
          v-bind:key="uid"
        >
          {{ value.displayName }}
        </span>
      </span>
      <span v-if="Object.keys(reviewed(item)).length > 0">
        Reviewed:
        <span
          class="label"
          v-for="(value, uid) in reviewed(item)"
          v-bind:key="uid"
        >
          {{ value.displayName }}
        </span>
      </span>
    </template>
-->
