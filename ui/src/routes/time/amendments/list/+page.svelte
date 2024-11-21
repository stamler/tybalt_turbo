<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { PageData } from "./$types";
  import type { TimeAmendmentsResponse } from "$lib/pocketbase-types";

  let { data }: { data: PageData } = $props();
  let items = $state(data.items);

  function hoursString(item: TimeAmendmentsResponse) {
    const hoursArray = [];
    if (item.hours) hoursArray.push(item.hours + " hrs");
    if (item.meals_hours) hoursArray.push(item.meals_hours + " hrs meals");
    return hoursArray.join(" + ");
  }

  async function del(id: string): Promise<void> {
    // return immediately if items is not an array
    if (!Array.isArray(items)) return;

    try {
      await pb.collection("time_amendments").delete(id);

      // remove the item from the list
      items = items.filter((item) => item.id !== id);
    } catch (error: any) {
      alert(error.data.message);
    }
  }
</script>

{#snippet anchor(item: TimeAmendmentsResponse)}{item.date}{/snippet}

{#snippet headline({ expand }: TimeAmendmentsResponse)}
  <span>
    {expand.uid.expand?.profiles_via_uid.given_name}
    {expand.uid.expand?.profiles_via_uid.surname}
  </span>
  -
  {#if expand?.time_type.code === "R"}
    <span>{expand.division.name}</span>
  {:else}
    <span>{expand?.time_type.name}</span>
  {/if}
{/snippet}

{#snippet byline({ expand, payout_request_amount }: TimeAmendmentsResponse)}
  {#if expand?.time_type.code === "OTO"}
    <span>${payout_request_amount}</span>
  {/if}
{/snippet}

{#snippet line1({ expand, job }: TimeAmendmentsResponse)}
  {#if expand?.time_type !== undefined && ["R", "RT"].includes(expand.time_type.code) && job !== ""}
    <span class="flex items-center gap-1">
      {expand?.job.number} - {expand?.job.description}
      {#if expand?.category !== undefined}
        <DsLabel color="teal">{expand?.category.name}</DsLabel>
      {/if}
    </span>
  {/if}
{/snippet}

{#snippet line2(item: TimeAmendmentsResponse)}{hoursString(item)}{/snippet}

{#snippet line3({ work_record, description }: TimeAmendmentsResponse)}
  {#if work_record !== ""}
    <span><span class="opacity-50">Work Record</span> {work_record} / </span>
  {/if}
  <span class="opacity-50">{description}</span>
{/snippet}

{#snippet actions({ id, committed }: TimeAmendmentsResponse)}
  {#if !committed}
    <DsActionButton
      action={`/time/amendments/${id}/edit`}
      icon="mdi:edit-outline"
      title="Edit"
      color="blue"
    />
    <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
  {/if}
{/snippet}

{#snippet groupHeader(field: string)}
  Week Ending {field}
{/snippet}

<DsList
  items={items as TimeAmendmentsResponse[]}
  search={true}
  inListHeader="Time Amendments"
  groupField="committed_week_ending"
  {groupHeader}
  {anchor}
  {headline}
  {byline}
  {line1}
  {line2}
  {line3}
  {actions}
/>
