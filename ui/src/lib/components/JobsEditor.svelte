<script lang="ts">
  import { flatpickrAction } from "$lib/utilities";
  import { globalStore } from "$lib/stores/global";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import { goto } from "$app/navigation";
  import type { JobsPageData } from "$lib/svelte-types";
  import DsActionButton from "./DSActionButton.svelte";

  let { data }: { data: JobsPageData } = $props();

  let errors = $state({} as any);
  let item = $state(data.item);
  let categories = $state(data.categories);

  async function save(event: Event) {
    event.preventDefault();

    try {
      if (data.editing && data.id !== null) {
        await pb.collection("jobs").update(data.id, item);
      } else {
        await pb.collection("jobs").create(item);
      }

      errors = {};
      goto("/jobs/list");
    } catch (error: any) {
      errors = error.data.data;
    }
  }
</script>

<svelte:head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css" />
</svelte:head>

<form
  class="flex w-full flex-col items-center gap-2 p-2"
  enctype="multipart/form-data"
  onsubmit={save}
>
  <span class="flex w-full flex-col gap-2 {errors.date !== undefined ? 'bg-red-200' : ''}">
    <label for="date">Date</label>
    <input
      class="flex-1"
      type="text"
      name="date"
      placeholder="Date"
      use:flatpickrAction
      bind:value={item.date}
    />
    {#if errors.date !== undefined}
      <span class="text-red-600">{errors.date.message}</span>
    {/if}
  </span>

  <!-- <DsSelector
    bind:value={item.job_division as string}
    items={$globalStore.divisions}
    {errors}
    fieldName="division"
    uiName="Division"
  >
    {#snippet optionTemplate(item)}
      {item.code} - {item.name}
    {/snippet}
  </DsSelector> -->

  <DsTextInput bind:value={item.number as string} {errors} fieldName="number" uiName="Number" />

  <DsTextInput
    bind:value={item.description as string}
    {errors}
    fieldName="description"
    uiName="Description"
  />

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton type="submit">Save</DsActionButton>
      <DsActionButton action="/jobs/list">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>

<!-- Create a new job (from the list page) -->
<!-- <form class="flex w-full flex-col items-center gap-2 p-2">
  <DsTextInput bind:value={item.number as string} {errors} fieldName="number" uiName="Job Number" />
  <DsTextInput
    bind:value={item.description as string}
    {errors}
    fieldName="description"
    uiName="Description"
  />
  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton action={save}>Save</DsActionButton>
      <DsActionButton action={clearForm}>Clear</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form> -->
