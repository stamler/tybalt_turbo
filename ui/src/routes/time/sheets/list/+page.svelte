<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { globalStore } from "$lib/stores/global";
  import { goto } from "$app/navigation";
  import ShareModal from "$lib/components/ShareModal.svelte";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import { shortDate } from "$lib/utilities";
  import type { TimeSheetTallyQueryRow } from "$lib/utilities";
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
      globalStore.refresh("time_sheets_tallies");

      // navigate to the time entries list to show the unbundled time entries
      goto(`/time/entries/list`);
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "An unknown error occurred");
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
      globalStore.refresh("time_sheets_tallies");
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "An unknown error occurred");
    }
  }

  function openRejectModal(id: string) {
    rejectModal?.openModal(id);
  }
</script>

{#snippet anchor({ week_ending }: TimeSheetTallyQueryRow)}
  <a class="font-bold hover:underline" href="/time/sheets/details">{shortDate(week_ending)}</a>
{/snippet}
{#snippet headline(tally: TimeSheetTallyQueryRow)}
  {#if tally.work_total_hours > 0}
    <span>{tally.work_total_hours} hours worked</span>
  {:else}
    <span>no work</span>
  {/if}
{/snippet}
{#snippet byline(tally: TimeSheetTallyQueryRow)}
  {#if tally.off_week_dates.length > 0}
    <span>/ off rotation week</span>
  {:else}
    <span>/ {tally.non_work_total_hours} hours off</span>
    {#if tally.off_rotation_dates.length > 0}
      <span>/ {tally.off_rotation_dates.length} day(s) off rotation</span>
    {/if}
    {#if tally.bank_entry_dates.length > 0}
      <span>/ {tally.rb_hours} hours banked</span>
    {/if}
  {/if}
{/snippet}
{#snippet line2(tally: TimeSheetTallyQueryRow)}
  <span>{tally.divisions.join(", ")}</span>
{/snippet}
{#snippet line1(tally: TimeSheetTallyQueryRow)}
  <span>{tally.job_numbers.join(", ")}</span>
{/snippet}
{#snippet line3(tally: TimeSheetTallyQueryRow)}
  {#if tally.rejected}
    <span class="text-red-600">Rejected: {tally.rejection_reason}</span>
  {/if}
  {#if tally.payout_request_dates.length > 0}
    <span>$${tally.payout_request_amount.toFixed(2)} in payout requests</span>
  {:else}
    <span>no payout requests</span>
  {/if}
  <!-- TODO: implement viewers, reviewed -->
  <span>Viewers, Reviewed</span>
{/snippet}
{#snippet actions({ id, approved }: TimeSheetTallyQueryRow)}
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
