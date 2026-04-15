<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import TimesheetSharedBadge from "$lib/components/TimesheetSharedBadge.svelte";
  import type { PageData } from "./$types";
  import type { TimeEntriesResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";
  import { shortDate } from "$lib/utilities";
  import { type UnsubscribeFunc } from "pocketbase";
  import { onMount, onDestroy, untrack } from "svelte";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import UncommitConfirmPopover from "$lib/components/UncommitConfirmPopover.svelte";
  import { goto } from "$app/navigation";
  import DsEditingDisabledBanner from "$lib/components/DsEditingDisabledBanner.svelte";
  import { timeEditingDisabledMessage, timeEditingEnabled } from "$lib/stores/appConfig";
  import {
    canApproveTimesheet,
    canCommitTimesheet,
    canRecallTimesheet,
    canRejectTimesheet,
  } from "$lib/timesheets/actions";
  import { getApiErrorMessage } from "$lib/errors";

  let { data }: { data: PageData } = $props();
  let items = $state(untrack(() => data.items));
  let tallies = $state(untrack(() => data.tallies));
  let timeSheet = $state(untrack(() => data.timeSheet));
  let approverInfo = $state(untrack(() => data.approverInfo as any));
  let committerInfo = $state(untrack(() => data.committerInfo));
  let sharedReviewerCount = $state(untrack(() => data.sharedReviewerCount ?? 0));
  let showUncommitConfirm = $state(false);
  let uncommitSubmitting = $state(false);
  let uncommitError = $state<string | null>(null);
  let rejectModal: RejectModal;
  const viewerId = pb.authStore.record?.id ?? "";
  const isCommitUser = () => $globalStore.showAllUi || $globalStore.claims.includes("commit");
  const isAdminUser = () => $globalStore.showAllUi || $globalStore.claims.includes("admin");

  async function refreshDetails(id: string) {
    try {
      const response = await pb.send(`/api/time_sheets/${id}/details`, {
        method: "GET",
      });
      timeSheet = response.timeSheet;
      items = response.items;
      approverInfo = response.approverInfo;
      committerInfo = { committer_name: response.approverInfo?.committer_name || "" };
      sharedReviewerCount = response.sharedReviewerCount ?? 0;
    } catch (error: any) {
      globalStore.addError(getApiErrorMessage(error, "Refresh failed"));
    }
  }

  // Subscribe to time entries changes for this specific time sheet
  let unsubscribeFunc: UnsubscribeFunc | undefined;
  onMount(async () => {
    if (items === undefined) {
      return;
    }
    try {
      unsubscribeFunc = await pb.collection("time_entries").subscribe<TimeEntriesResponse>(
        "*",
        async (e) => {
          if (!Array.isArray(items)) return;
          switch (e.action) {
            case "create":
              if (e.record.tsid === data.timesheetId) {
                items = [e.record, ...items];
              }
              break;
            case "update":
              items = items.map((item) => (item.id === e.record.id ? e.record : item));
              break;
            case "delete":
              items = items.filter((item) => item.id !== e.record.id);
              break;
          }
        },
        {
          filter: pb.filter("tsid={:tsid}", { tsid: data.timesheetId }),
          expand: "job,time_type,division,category,role",
        },
      );
    } catch {
      unsubscribeFunc = undefined;
    }
  });

  onDestroy(async () => {
    unsubscribeFunc?.();
  });

  function hoursString(item: TimeEntriesResponse) {
    const hoursArray = [];
    if (item.hours) hoursArray.push(item.hours + " hrs");
    if (item.meals_hours) hoursArray.push(item.meals_hours + " hrs meals");
    return hoursArray.join(" + ");
  }

  async function del(id: string): Promise<void> {
    try {
      await pb.collection("time_entries").delete(id);
    } catch (error: any) {
      globalStore.addError(getApiErrorMessage(error, "Delete failed"));
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
      await refreshDetails(id);
    } catch (error: any) {
      globalStore.addError(getApiErrorMessage(error, "Approve failed"));
    }
  }

  async function commit(id: string) {
    try {
      await pb.send(`/api/time_sheets/${id}/commit`, {
        method: "POST",
      });
      await refreshDetails(id);
    } catch (error: any) {
      globalStore.addError(getApiErrorMessage(error, "Commit failed"));
    }
  }

  async function uncommit(id: string) {
    uncommitSubmitting = true;
    uncommitError = null;
    try {
      await pb.send(`/api/time_sheets/${id}/uncommit`, {
        method: "POST",
      });
      showUncommitConfirm = false;
      await refreshDetails(id);
    } catch (error: any) {
      uncommitError = getApiErrorMessage(error, "Uncommit failed");
    } finally {
      uncommitSubmitting = false;
    }
  }

  async function recall(id: string) {
    try {
      await pb.send(`/api/time_sheets/${id}/unbundle`, {
        method: "POST",
      });

      // navigate to the time entries list to show the unbundled time entries
      goto(`/time/entries/list`);
    } catch (error: any) {
      globalStore.addError(getApiErrorMessage(error, "Recall failed"));
    }
  }

  function openRejectModal(recordId: string) {
    rejectModal?.openModal(recordId);
  }

  function openUncommitConfirm() {
    uncommitError = null;
    showUncommitConfirm = true;
  }

  function closeUncommitConfirm() {
    uncommitError = null;
    showUncommitConfirm = false;
  }
</script>

<div class="mx-auto p-4">
  <h1 class="mb-4 text-2xl font-bold">Time Sheet Details</h1>
  {#if !$timeEditingEnabled}
    <DsEditingDisabledBanner message={timeEditingDisabledMessage} />
  {/if}
  <div class="mb-4 space-y-2">
    <h2 class="text-lg font-semibold">Week Ending: {shortDate(timeSheet.week_ending, true)}</h2>
    <div class="space-y-4 rounded-lg bg-neutral-100 p-4">
      <div class="space-y-2 text-gray-600">
        {#if timeSheet.approved && timeSheet.rejected === ""}
          <div class="flex items-center gap-1">
            <DsLabel color="green">Approved</DsLabel>
            {#if approverInfo.approver_name}
              <span>by {approverInfo.approver_name}</span>
            {/if}
            <span>on {shortDate(timeSheet.approved.split("T")[0])}</span>
          </div>
        {/if}

        {#if timeSheet.committed && timeSheet.rejected === ""}
          <div class="mt-1 flex items-center gap-1">
            <DsLabel color="blue">Committed</DsLabel>
            {#if committerInfo.committer_name}
              <span>by {committerInfo.committer_name}</span>
            {/if}
            <span>on {shortDate(timeSheet.committed.split("T")[0])}</span>
          </div>
        {/if}

        {#if timeSheet.rejected !== ""}
          <div class="mt-1 flex items-center gap-1">
            <DsLabel color="red">Rejected</DsLabel>
            {#if approverInfo.rejector_name}
              <span>by {approverInfo.rejector_name}</span>
            {/if}
            <span>on {shortDate(timeSheet.rejected.split("T")[0])}</span>
          </div>
          <div class="mt-1 text-red-600 italic">
            {timeSheet.rejection_reason}
          </div>
        {:else if !timeSheet.committed && !timeSheet.approved && timeSheet.rejected === ""}
          <DsLabel color="orange">Pending</DsLabel>
        {/if}

        {#if sharedReviewerCount > 0}
          <div class="flex flex-wrap items-center gap-1">
            <TimesheetSharedBadge count={sharedReviewerCount} />
          </div>
        {/if}
      </div>
      <!-- Action Buttons -->
      <div class="flex flex-wrap gap-2 empty:hidden">
        {#if timeSheet.committed !== "" && isAdminUser()}
          <DsActionButton
            action={openUncommitConfirm}
            icon="mdi:undo"
            title="Uncommit"
            color="orange"
          />
        {:else if timeSheet.rejected !== ""}
          <!-- Rejected: allow recall -->
          {#if canRecallTimesheet(timeSheet, viewerId)}
            <DsActionButton
              action={() => recall(timeSheet.id)}
              icon="mdi:rewind"
              title="Recall"
              color="orange"
            />
          {/if}
        {:else if timeSheet.approved === ""}
          <!-- Pending: recall, approve, reject -->
          {#if canRecallTimesheet(timeSheet, viewerId)}
            <DsActionButton
              action={() => recall(timeSheet.id)}
              icon="mdi:rewind"
              title="Recall"
              color="orange"
            />
          {/if}
          {#if canApproveTimesheet(timeSheet, viewerId)}
            <DsActionButton
              action={() => approve(timeSheet.id)}
              icon="mdi:approve"
              title="Approve"
              color="green"
            />
          {/if}
          {#if canRejectTimesheet(timeSheet, viewerId, isCommitUser())}
            <DsActionButton
              action={() => openRejectModal(timeSheet.id)}
              icon="mdi:cancel"
              title="Reject"
              color="orange"
            />
          {/if}
        {:else if timeSheet.approved !== "" && timeSheet.committed === ""}
          <!-- Approved (not committed yet): commit, reject -->
          {#if canCommitTimesheet(timeSheet, isCommitUser())}
            <DsActionButton
              action={() => commit(timeSheet.id)}
              icon="mdi:check-all"
              title="Commit"
              color="green"
            />
          {/if}
          {#if canRejectTimesheet(timeSheet, viewerId, isCommitUser())}
            <DsActionButton
              action={() => openRejectModal(timeSheet.id)}
              icon="mdi:cancel"
              title="Reject"
              color="orange"
            />
          {/if}
        {/if}
      </div>
    </div>
  </div>

  <!-- Tallies Summary -->
  <div class="mb-4 rounded-lg bg-neutral-100 p-4">
    <h3 class="mb-2 text-lg font-semibold">Summary</h3>
    <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
      <div>
        <div class="font-medium">Work Hours</div>
        <div class="text-2xl font-bold text-blue-600">
          {tallies.workHoursTally.total}
        </div>
        <div class="text-sm text-gray-600">
          {tallies.workHoursTally.jobHours} on jobs / {tallies.workHoursTally.hours} non-job
        </div>
      </div>
      <div>
        <div class="font-medium">Off Hours</div>
        <div class="text-2xl font-bold text-orange-600">
          {tallies.nonWorkHoursTally.total}
        </div>
        <div class="text-sm text-gray-600">
          {#if tallies.offRotationDates.length > 0}
            {tallies.offRotationDates.length} day(s) off rotation
          {/if}
          {#if tallies.offWeek.length > 0}
            Full week off
          {/if}
        </div>
      </div>
      <div>
        <div class="font-medium">Meals Hours</div>
        <div class="text-2xl font-bold text-green-600">
          {tallies.mealsHoursTally}
        </div>
      </div>
    </div>

    {#if tallies.bankEntries.length > 0 || tallies.payoutRequests.length > 0}
      <div class="mt-4 border-t pt-4">
        {#if tallies.bankEntries.length === 1}
          <span class="mr-2 inline-block rounded-sm bg-blue-100 px-2 py-1 text-sm text-blue-800">
            {tallies.bankEntries[0].hours} hours banked
          </span>
        {:else if tallies.bankEntries.length > 1}
          <span class="mr-2 inline-block rounded-sm bg-red-100 px-2 py-1 text-sm text-red-800">
            Multiple bank entries
          </span>
        {/if}
        {#if tallies.payoutRequests.length === 1}
          <span class="inline-block rounded-sm bg-green-100 px-2 py-1 text-sm text-green-800">
            ${tallies.payoutRequests[0].payout_request_amount} payout requested
          </span>
        {:else if tallies.payoutRequests.length > 1}
          <span class="inline-block rounded-sm bg-red-100 px-2 py-1 text-sm text-red-800">
            Multiple payout requests
          </span>
        {/if}
      </div>
    {/if}

    {#if Object.keys(tallies.jobsTally).length > 0}
      <div class="mt-4 border-t pt-4">
        <div class="mb-2 font-medium">Jobs</div>
        <div class="flex flex-wrap gap-2">
          {#each Object.entries(tallies.jobsTally) as [jobNumber, hours]}
            <DsLabel color="blue">{jobNumber}: {hours}h</DsLabel>
          {/each}
        </div>
      </div>
    {/if}

    {#if Object.keys(tallies.divisionsTally).length > 0}
      <div class="mt-4 border-t pt-4">
        <div class="mb-2 font-medium">Divisions</div>
        <div class="flex flex-wrap gap-2">
          {#each Object.entries(tallies.divisionsTally) as [divisionCode, divisionName]}
            <DsLabel color="teal">{divisionCode}: {divisionName}</DsLabel>
          {/each}
        </div>
      </div>
    {/if}
  </div>

  <!-- Time Entries List -->
  <div class="overflow-hidden rounded-lg">
    <DsList items={items as TimeEntriesResponse[]} search={true} inListHeader="Time Entries">
      {#snippet anchor(item: TimeEntriesResponse)}
        {item.date}
      {/snippet}

      {#snippet headline({ expand }: TimeEntriesResponse)}
        {#if expand?.time_type.code === "R"}
          <span>{expand.division.name}</span>
        {:else}
          <span>{expand?.time_type.name}</span>
        {/if}
      {/snippet}

      {#snippet byline({ expand, payout_request_amount }: TimeEntriesResponse)}
        {#if expand?.time_type.code === "OTO"}
          <span>${payout_request_amount}</span>
        {/if}
      {/snippet}

      {#snippet line1({ expand, job }: TimeEntriesResponse)}
        {#if expand?.time_type !== undefined && ["R", "RT"].includes(expand.time_type.code) && job !== ""}
          <span class="flex items-center gap-1">
            {expand?.job.number} - {expand?.job.description}
            {#if expand?.category !== undefined}
              <DsLabel color="teal">{expand?.category.name}</DsLabel>
            {/if}
            {#if expand?.role !== undefined}
              <DsLabel color="purple">{expand?.role.name}</DsLabel>
            {/if}
          </span>
        {/if}
      {/snippet}

      {#snippet line2(item: TimeEntriesResponse)}
        {hoursString(item)}
      {/snippet}

      {#snippet line3({ work_record, description }: TimeEntriesResponse)}
        {#if work_record !== ""}
          <span><span class="opacity-50">Work Record</span> {work_record} / </span>
        {/if}
        <span class="opacity-50">{description}</span>
      {/snippet}
    </DsList>
  </div>

  <!-- Reject Modal -->
  <RejectModal
    collectionName="time_sheets"
    bind:this={rejectModal}
    on:refresh={() => {
      refreshDetails(timeSheet.id);
    }}
  />
  <UncommitConfirmPopover
    bind:show={showUncommitConfirm}
    recordLabel="time sheet"
    submitting={uncommitSubmitting}
    error={uncommitError}
    onSubmit={() => uncommit(timeSheet.id)}
    onCancel={closeUncommitConfirm}
  />
</div>
