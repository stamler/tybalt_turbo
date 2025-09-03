<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { goto } from "$app/navigation";
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

  async function del(id: string): Promise<void> {
    if (!Array.isArray(items)) return;
    try {
      await pb.collection("time_amendments").delete(id);
      items = items.filter((item) => item.id !== id);
    } catch (error: any) {
      alert(error.data.message);
    }
  }

  async function commit(id: string): Promise<void> {
    try {
      await pb.send(`/api/time_amendments/${id}/commit`, {
        method: "POST",
      });
      goto("/time/amendments/pending");
    } catch (error: any) {
      alert(error?.response?.error ?? error?.data?.message ?? "Error committing record");
    }
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

{#snippet actions({ id }: TimeAmendmentsAugmentedResponse)}
  <DsActionButton
    action={`/time/amendments/${id}/edit`}
    icon="mdi:edit-outline"
    title="Edit"
    color="blue"
  />
  <DsActionButton action={() => commit(id)} icon="mdi:check-all" title="Commit" color="green" />
  <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
{/snippet}

<DsList
  items={items as TimeAmendmentsAugmentedResponse[]}
  search={true}
  inListHeader="Pending Time Amendments"
  groupField="uid_name"
  {anchor}
  {headline}
  {byline}
  {line1}
  {line2}
  {line3}
  {actions}
/>
