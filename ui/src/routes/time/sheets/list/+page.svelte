<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { globalStore } from "$lib/stores/global";
  import { goto } from "$app/navigation";
  import ShareModal from "$lib/components/ShareModal.svelte";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import { shortDate, hoursWorked, jobs, divisions, payoutRequests } from "$lib/utilities";
  import type { TimeSheetTally } from "$lib/utilities";
  import type { SvelteComponent } from "svelte";
  let shareModal: SvelteComponent;
  let rejectModal: SvelteComponent;

  async function unbundle(id: string) {
    try {
      await pb.send(`/api/time_sheets/${id}/unbundle`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      // refresh the time sheets list in the global store
      globalStore.refresh("time_sheets");

      // navigate to the time entries list to show the unbundled time entries
      goto(`/time/entries/list`);
    } catch (error) {
      globalStore.addError(error?.response.error);
    }
  }

  async function approve(id: string) {
    try {
      await pb.send(`/api/time_sheets/${id}/approve`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      // refresh the time sheets list in the global store
      globalStore.refresh("time_sheets");
    } catch (error) {
      globalStore.addError(error?.response.error);
    }
  }

  function openRejectModal(id: string) {
    rejectModal?.openModal(id);
  }
</script>

{#snippet anchor({ week_ending }: TimeSheetTally)}
  <a class="font-bold hover:underline" href="/time/sheets/details">{shortDate(week_ending)}</a>
{/snippet}
{#snippet headline(tally: TimeSheetTally)}
  <span>{hoursWorked(tally)}</span>
{/snippet}
{#snippet byline(tally: TimeSheetTally)}
  {#if tally.offWeek.length > 0}
    <span>/ off rotation week</span>
  {:else}
    <span>/ {tally.nonWorkHoursTally.total} hours off</span>
    {#if tally.offRotationDates.length > 0}
      <span>/ {tally.offRotationDates.length} day(s) off rotation</span>
    {/if}
    {#if tally.bankEntries.length > 0}
      <span>/ {tally.bankEntries.reduce((sum, entry) => sum + entry.hours, 0)} hours banked</span>
    {/if}
  {/if}
{/snippet}
{#snippet line2(tally: TimeSheetTally)}
  <span>{divisions(tally)}</span>
{/snippet}
{#snippet line1(tally: TimeSheetTally)}
  <span>{jobs(tally)}</span>
{/snippet}
{#snippet line3(tally: TimeSheetTally)}
  {#if tally.rejected}
    <span class="text-red-600">Rejected: {tally.rejection_reason}</span>
  {/if}
  {#if tally.payoutRequests.length > 0}
    <span>{payoutRequests(tally)}</span>
  {/if}
  <!-- TODO: implement viewers, reviewed -->
  <span>Viewers, Reviewed</span>
{/snippet}
{#snippet actions({ id, approved }: TimeSheetTally)}
  <DsActionButton action={() => unbundle(id)} icon="mdi:rewind" title="Recall" color="orange" />
  {#if approved === ""}
    <DsActionButton action={() => approve(id)} icon="mdi:approve" title="Approve" color="green" />
  {/if}
  <DsActionButton
    action={() => openRejectModal(id)}
    icon="mdi:cancel"
    title="Reject"
    color="orange"
  />
  <DsActionButton
    action={() => shareModal?.openModal(id)}
    icon="mdi:ios-share"
    title="Share with another manager"
    color="purple"
  />
{/snippet}

<ShareModal bind:this={shareModal} collectionName="time_sheet_reviewers" />
<RejectModal collectionName="time_sheets" bind:this={rejectModal} />

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
