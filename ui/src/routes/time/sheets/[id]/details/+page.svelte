<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { PageData } from "./$types";
  import type { TimeEntriesResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";
  import { shortDate } from "$lib/utilities";
  import { type UnsubscribeFunc } from "pocketbase";
  import { onMount, onDestroy } from "svelte";

  let { data }: { data: PageData } = $props();
  let items = $state(data.items);
  let tallies = $state(data.tallies);
  let timeSheet = $state(data.timeSheet);
  let approverInfo = $state(data.approverInfo);

  // Subscribe to time entries changes for this specific time sheet
  let unsubscribeFunc: UnsubscribeFunc;
  onMount(async () => {
    if (items === undefined) {
      return;
    }
    unsubscribeFunc = await pb.collection("time_entries").subscribe<TimeEntriesResponse>(
      "*",
      async (e) => {
        // return immediately if items is not an array
        if (!Array.isArray(items)) return;
        switch (e.action) {
          case "create":
            // Only add if it belongs to this time sheet
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
        expand: "job,time_type,division,category",
      },
    );
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
      globalStore.addError(error?.response?.message);
    }
  }
</script>

<div class="container mx-auto p-4">
  <h1 class="mb-4 text-2xl font-bold">Time Sheet Details</h1>
  <div class="mb-4">
    <h2 class="text-lg font-semibold">Week Ending: {shortDate(timeSheet.week_ending, true)}</h2>
    <div class="text-gray-600">
      Status: {#if timeSheet.approved}
        <span class="font-medium text-green-600">Approved</span>
        {#if approverInfo.approver_name}
          by {approverInfo.approver_name}
          on {shortDate(timeSheet.approved.split("T")[0])}
        {/if}
      {:else}
        <span class="font-medium text-orange-600">Pending</span>
      {/if}
    </div>
  </div>

  <!-- Tallies Summary -->
  <div class="mb-6 rounded-lg bg-gray-50 p-4">
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
          <span class="mr-2 inline-block rounded bg-blue-100 px-2 py-1 text-sm text-blue-800">
            {tallies.bankEntries[0].hours} hours banked
          </span>
        {:else if tallies.bankEntries.length > 1}
          <span class="mr-2 inline-block rounded bg-red-100 px-2 py-1 text-sm text-red-800">
            Multiple bank entries
          </span>
        {/if}
        {#if tallies.payoutRequests.length === 1}
          <span class="inline-block rounded bg-green-100 px-2 py-1 text-sm text-green-800">
            ${tallies.payoutRequests[0].payout_request_amount} payout requested
          </span>
        {:else if tallies.payoutRequests.length > 1}
          <span class="inline-block rounded bg-red-100 px-2 py-1 text-sm text-red-800">
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

    {#snippet actions({ id }: TimeEntriesResponse)}
      <DsActionButton
        action={`/time/entries/${id}/edit`}
        icon="mdi:edit-outline"
        title="Edit"
        color="blue"
      />
      <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
    {/snippet}
  </DsList>
</div>
