<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { authStore } from "$lib/stores/auth";
  import { get } from "svelte/store";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { PageData } from "./$types";
  import type { TimeEntriesResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";
  import { goto } from "$app/navigation";
  import { calculateTally } from "$lib/utilities";
  import { type UnsubscribeFunc } from "pocketbase";
  import { onMount, onDestroy } from "svelte";
  let { data }: { data: PageData } = $props();
  let items = $state(data.items);

  // Subscribe to the base collection but update the items from the augmented
  // view
  let unsubscribeFunc: UnsubscribeFunc;
  onMount(async () => {
    if (items === undefined) {
      return;
    }
    const userId = get(authStore)?.model?.id || "";

    unsubscribeFunc = await pb.collection("time_entries").subscribe<TimeEntriesResponse>(
      "*",
      async (e) => {
        // return immediately if items is not an array
        if (!Array.isArray(items)) return;
        switch (e.action) {
          case "create":
            items = [e.record, ...items];
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
        filter: pb.filter("tsid='' && uid={:userId}", { userId }),
        expand: "job,time_type,division,category",
      },
    );
  });
  onDestroy(async () => {
    unsubscribeFunc();
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

  async function copyToTomorrow(id: string): Promise<void> {
    try {
      await pb.send(`/api/time_entries/${id}/copy_to_tomorrow`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });
    } catch (error: any) {
      globalStore.addError(error?.response?.message);
    }
  }

  async function bundle(weekEnding: string) {
    try {
      await pb.send(`/api/time_sheets/${weekEnding}/bundle`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      // navigate to the time sheets list to show the bundled time sheets
      goto(`/time/sheets/list`);
    } catch (error: any) {
      globalStore.addError(error?.response?.message);
    }
  }
</script>

{#snippet anchor(item: TimeEntriesResponse)}{item.date}{/snippet}

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

{#snippet line2(item: TimeEntriesResponse)}{hoursString(item)}{/snippet}

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
  <DsActionButton
    action={() => copyToTomorrow(id)}
    icon="mdi:content-copy"
    title="Copy to tomorrow"
    color="green"
  />
  <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
{/snippet}

{#snippet groupHeader(field: string)}
  Week Ending {field}
{/snippet}

{#snippet groupFooter(groupKey: string, items: TimeEntriesResponse[])}
  <div class="flex items-center justify-center px-4 py-2">Totals</div>
  <div class="flex flex-col py-2">
    {#if Array.isArray(items)}
      {@const tallies = calculateTally(items)}
      <!--
        Daily Hours Totals (GitHub Issue #98)
        
        This implementation shows daily totals in the weekly footer for simplicity,
        avoiding modifications to the DSList component.
        
        Alternative approaches for future consideration:
        1. Add subgrouping to DSList (subGroupField, subGroupFooter props) to show
           daily totals inline after each day's entries within the list
        2. Create a specialized TimeEntriesList component with built-in date-based
           subtotaling
        3. Inject synthetic "total" rows into the data array (not recommended due
           to type safety concerns)
      -->
      {@const dayNames = ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"]}
      {@const dailyTotals = Object.entries(
        items.reduce(
          (acc, item) => {
            const hours = (item.hours || 0) + (item.meals_hours || 0);
            acc[item.date] = (acc[item.date] || 0) + hours;
            return acc;
          },
          {} as Record<string, number>,
        ),
      )
        .map(([date, total]) => ({
          day: dayNames[new Date(date + "T00:00:00").getDay()],
          total,
          date,
        }))
        .filter(({ total }) => total > 0)
        .sort((a, b) => a.date.localeCompare(b.date))}
      <div class="mb-1 flex flex-wrap gap-x-3 gap-y-1 text-sm">
        {#each dailyTotals as { day, total }}
          <span class="whitespace-nowrap">
            <span class="opacity-50">{day}:</span>
            {total}h
          </span>
        {/each}
      </div>
      <div class="headline_wrapper">
        <div class="headline">
          {tallies.workHoursTally.total + tallies.nonWorkHoursTally.total} hours
        </div>
        <div class="byline">
          {tallies.workHoursTally.total} worked /
          {tallies.nonWorkHoursTally.total} off
        </div>
      </div>
      <div class="firstline">
        {#if tallies.workHoursTally.jobHours > 0}
          <span>{tallies.workHoursTally.jobHours} hours on jobs</span>
        {/if}
        {#if tallies.workHoursTally.hours > 0}
          <span>{tallies.workHoursTally.hours} non-job hours</span>
        {/if}
        {#if tallies.mealsHoursTally > 0}
          <span>{tallies.mealsHoursTally} hours meals</span>
        {/if}
      </div>
      <div class="secondline">
        {#if tallies.offRotationDates.length > 0}
          <span>{tallies.offRotationDates.length} day(s) off rotation</span>
        {/if}
        {#if tallies.offWeek.length > 0}
          <span>Full Week off</span>
        {/if}
      </div>
      <div class="thirdline">
        {#if tallies.bankEntries.length === 1}
          <span>{tallies.bankEntries[0].hours} hours banked</span>
        {:else if tallies.bankEntries.length > 1}
          <span class="rounded bg-red-200 p-1 text-red-600">
            More than one banked time entry exists.
          </span>
        {/if}
        {#if tallies.payoutRequests.length === 1}
          <span>${tallies.payoutRequests[0].payout_request_amount} payout requested</span>
        {:else if tallies.payoutRequests.length > 1}
          <span class="rounded bg-red-200 p-1 text-red-600">
            More than one payout request entry exists.
          </span>
        {/if}
      </div>
    {/if}
  </div>
  <div class="flex items-center gap-1 px-2 py-2">
    <DsActionButton
      action={() => bundle(groupKey)}
      icon="mdi:box-up"
      title="Submit"
      color="purple"
    />
  </div>
{/snippet}
<DsList
  items={items as TimeEntriesResponse[]}
  search={true}
  inListHeader="Time Entries"
  groupField="week_ending"
  {groupHeader}
  {groupFooter}
  {anchor}
  {headline}
  {byline}
  {line1}
  {line2}
  {line3}
  {actions}
/>
