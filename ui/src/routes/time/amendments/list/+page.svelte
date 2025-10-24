<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import type { PageData } from "./$types";
  import type { TimeAmendmentsAugmentedResponse } from "$lib/pocketbase-types";

  let { data }: { data: PageData } = $props();
  let items = $state(data.items);

  function hoursString(item: TimeAmendmentsAugmentedResponse) {
    const hoursArray = [];
    if (item.hours) hoursArray.push(item.hours + " hrs");
    if (item.meals_hours) hoursArray.push(item.meals_hours + " hrs meals");
    return hoursArray.join(" + ");
  }
</script>

{#snippet anchor(item: TimeAmendmentsAugmentedResponse)}{item.date}{/snippet}

{#snippet headline({
  uid_name,
  time_type_name,
  time_type_code,
  division_name,
}: TimeAmendmentsAugmentedResponse)}
  <span>
    {uid_name}
  </span>
  -
  {#if time_type_code === "R"}
    <span>{division_name}</span>
  {:else}
    <span>{time_type_name}</span>
  {/if}
{/snippet}

{#snippet byline({ time_type_code, payout_request_amount }: TimeAmendmentsAugmentedResponse)}
  {#if time_type_code === "OTO"}
    <span>${payout_request_amount}</span>
  {/if}
{/snippet}

{#snippet line1({
  time_type_code,
  job_number,
  job_description,
  category_name,
  category,
}: TimeAmendmentsAugmentedResponse)}
  {#if time_type_code !== undefined && ["R", "RT"].includes(time_type_code) && job_number !== ""}
    <span class="flex items-center gap-1">
      {job_number} - {job_description}
      {#if category !== ""}
        <DsLabel color="teal">{category_name}</DsLabel>
      {/if}
    </span>
  {/if}
{/snippet}

{#snippet line2(item: TimeAmendmentsAugmentedResponse)}{hoursString(item)}{/snippet}

{#snippet line3({ work_record, description }: TimeAmendmentsAugmentedResponse)}
  {#if work_record !== ""}
    <span><span class="opacity-50">Work Record</span> {work_record} / </span>
  {/if}
  <span class="opacity-50">{description}</span>
{/snippet}

{#snippet actions({ committed }: TimeAmendmentsAugmentedResponse)}
  {#if committed}
    <DsLabel color="green">Committed</DsLabel>
  {/if}
{/snippet}

{#snippet groupHeader(field: string)}
  Week Ending {field}
{/snippet}

<DsList
  items={items as TimeAmendmentsAugmentedResponse[]}
  search={true}
  inListHeader="Time Amendments"
  groupField="committed_week_ending"
  groupSort="DESC"
  {groupHeader}
  {anchor}
  {headline}
  {byline}
  {line1}
  {line2}
  {line3}
  {actions}
/>
