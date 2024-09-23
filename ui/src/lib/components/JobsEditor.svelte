<script lang="ts">
  import { flatpickrAction } from "$lib/utilities";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
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
  let newCategory = $state("");

  async function addCategory() {
    if (newCategory.trim() === "") return;

    try {
      const category = await pb.collection("categories").create(
        {
          job: item.id,
          name: newCategory.trim(),
        },
        { returnRecord: true },
      );

      categories.push(category);
      newCategory = "";
    } catch (error: any) {
      errors.categories = error.data.data;
    }
  }

  async function removeCategory(categoryId: string) {
    try {
      await pb.collection("categories").delete(categoryId);
      categories = categories.filter((category) => category.id !== categoryId);
    } catch (error: any) {
      alert(error.data.message);
    }
  }

  function preventDefault(fn: (event: Event) => void) {
    return (event: Event) => {
      event.preventDefault();
      fn(event);
    };
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

  <div class="flex w-full flex-col gap-2 {errors.categories !== undefined ? 'bg-red-200' : ''}">
    <label for="categories">Categories</label>
    <div class="flex flex-wrap gap-1">
      {#each categories as category}
        <span class="flex items-center rounded-full bg-neutral-200 px-2">
          <span>{category.name}</span>
          <button
            class="text-neutral-500"
            onclick={preventDefault(() => removeCategory(category.id))}
          >
            &times;
          </button>
        </span>
      {/each}
    </div>
    <div class="flex items-center gap-1">
      <DsTextInput
        bind:value={newCategory as string}
        {errors}
        fieldName="newCategory"
        uiName="Add Category"
      />
      <DsActionButton
        action={addCategory}
        icon="feather:plus-circle"
        color="green"
        title="Add Category"
      />
    </div>
    {#if errors.categories !== undefined}
      <span class="text-red-600">{errors.categories.message}</span>
    {/if}
  </div>

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
