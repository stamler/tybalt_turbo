<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import type { TimeEntriesRecord } from "$lib/pocketbase-types";

  let { data }: { data: PageData } = $props();

  function hoursString(item: TimeEntriesRecord) {
    const hoursArray = [];
    if (item.hours) hoursArray.push(item.hours + " hrs");
    if (item.meals_hours) hoursArray.push(item.meals_hours + " hrs meals");
    return hoursArray.join(" + ");
  }

  async function del(id: string): Promise<void> {
    // return immediately if data.items is not an array
    if (!Array.isArray(data.items)) return;

    try {
      await pb.collection("time_entries").delete(id);

      // remove the item from the list
      data.items = data.items.filter((item) => item.id !== id);
    } catch (error: any) {
      alert(error.data.message);
    }
  }

  function calculateTallies(items: TimeEntriesRecord[]) {
    const tallies = {
      workHoursTally: { jobHours: 0, hours: 0, total: 0 },
      nonWorkHoursTally: { total: 0 } as Record<string, number>,
      mealsHoursTally: 0,
      bankEntries: [] as TimeEntriesRecord[],
      payoutRequests: [] as TimeEntriesRecord[],
      offRotationDates: [] as string[],
      offWeek: [] as string[],
    };

    items.forEach((item) => {
      const timeType = item.expand?.time_type.code;

      if (timeType === "R" || timeType === "RT") {
        if (item.job === "") {
          tallies.workHoursTally.hours += item.hours;
        } else {
          tallies.workHoursTally.jobHours += item.hours;
        }
        tallies.workHoursTally.total += item.hours;
        tallies.mealsHoursTally += item.meals_hours;
      } else if (timeType === "OR") {
        tallies.offRotationDates.push(item.date);
      } else if (timeType === "OW") {
        tallies.offWeek.push(item.date);
      } else if (timeType === "OTO") {
        tallies.payoutRequests.push(item);
      } else if (timeType === "RB") {
        tallies.bankEntries.push(item);
      } else {
        tallies.nonWorkHoursTally[timeType] =
          (tallies.nonWorkHoursTally[timeType] || 0) + item.hours;
        tallies.nonWorkHoursTally.total += item.hours;
      }
    });

    return tallies;
  }

  async function bundle(weekEnding: string) {
    // Implement bundling logic here
    console.log(`Bundling timesheet for week ending ${weekEnding}`);
    // You'll need to implement the actual bundling functionality
  }
</script>

{#snippet anchor(item)}{item.date}{/snippet}

{#snippet headline({ expand })}
  {#if expand?.time_type.code === "R"}
    <span>{expand.division.name}</span>
  {:else}
    <span>{expand?.time_type.name}</span>
  {/if}
{/snippet}

{#snippet byline({ expand, payout_request_amount })}
  {#if expand?.time_type.code === "OTO"}
    <span>${payout_request_amount}</span>
  {/if}
{/snippet}

{#snippet line1({ expand, job })}
  {#if ["R", "RT"].includes(expand?.time_type.code) && job !== ""}
    <span>{expand?.job.number} - {expand?.job.description}</span>
    {#if expand?.job.category}
      <span class="label">{expand.job.category}</span>
    {/if}
  {/if}
{/snippet}

{#snippet line2(item)}{hoursString(item)}{/snippet}

{#snippet line3({ work_record, description })}
  {#if work_record !== ""}
    <span><span class="opacity-50">Work Record</span> {work_record} / </span>
  {/if}
  <span class="opacity-50">{description}</span>
{/snippet}

{#snippet actions({ id })}
  <a href="/time/entries/{id}/edit">edit</a>
  <button type="button" onclick={() => del(id)}>delete</button>
{/snippet}

{#snippet groupHeader(field)}
  Week Ending {field}
{/snippet}

{#snippet groupFooter(groupKey, items)}
  <div class="flex items-center justify-center px-4 py-2">Totals</div>
  <div class="flex flex-col py-2">
    {#if Array.isArray(items)}
      {@const tallies = calculateTallies(items)}
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
    <button onclick={() => bundle(groupKey)}>bundle and submit</button>
  </div>
{/snippet}
<DsList
  items={data.items as TimeEntriesRecord[]}
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
