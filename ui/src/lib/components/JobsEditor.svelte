<script lang="ts">
  import { flatpickrAction, fetchContacts } from "$lib/utilities";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { goto } from "$app/navigation";
  import type { JobsPageData } from "$lib/svelte-types";
  import DsActionButton from "./DSActionButton.svelte";
  import { globalStore } from "$lib/stores/global";
  import type { ContactsResponse } from "$lib/pocketbase-types";
  let { data }: { data: JobsPageData } = $props();

  let errors = $state({} as any);
  let item = $state(data.item);
  let categories = $state(data.categories);
  let contacts = $state([] as ContactsResponse[]);

  let newCategory = $state("");
  let newCategories = $state([] as string[]);
  let categoriesToDelete = $state([] as string[]);

  // Watch for changes to the client and fetch contacts accordingly
  $effect(() => {
    if (item.client) {
      fetchContacts(item.client).then((c) => (contacts = c));
    }
  });

  async function save(event: Event) {
    event.preventDefault();

    try {
      let jobId = data.id;

      if (data.editing && jobId !== null) {
        await pb.collection("jobs").update(jobId, item);
      } else {
        const createdJob = await pb.collection("jobs").create(item);
        jobId = createdJob.id;
      }

      // Add new categories
      for (const categoryName of newCategories) {
        await pb.collection("categories").create(
          {
            job: jobId,
            name: categoryName.trim(),
          },
          { returnRecord: true },
        );
      }

      // Remove deleted categories
      for (const categoryId of categoriesToDelete) {
        await pb.collection("categories").delete(categoryId);
      }

      // reload jobs in the global store
      globalStore.refresh("jobs");

      errors = {};
      goto("/jobs/list");
    } catch (error: any) {
      errors = error.data.data;
    }
  }

  async function addCategory() {
    if (newCategory.trim() === "") return;

    newCategories.push(newCategory.trim());
    newCategory = "";
  }

  async function removeCategory(categoryId: string) {
    categoriesToDelete.push(categoryId);
    categories = categories.filter((category) => category.id !== categoryId);
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
  <span class="flex w-full gap-2 {errors.date !== undefined ? 'bg-red-200' : ''}">
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

  <DsSelector
    bind:value={item.manager as string}
    items={$globalStore.managers}
    {errors}
    fieldName="manager"
    uiName="Manager"
  >
    {#snippet optionTemplate(item)}
      {item.surname}, {item.given_name}
    {/snippet}
  </DsSelector>

  <DsTextInput
    bind:value={item.description as string}
    {errors}
    fieldName="description"
    uiName="Description"
  />

  {#if $globalStore.clientsIndex !== null}
    <DsAutoComplete
      bind:value={item.client as string}
      index={$globalStore.clientsIndex}
      {errors}
      fieldName="client"
      uiName="Client"
    >
      {#snippet resultTemplate(item)}{item.name}{/snippet}
    </DsAutoComplete>
  {/if}

  <!-- <DsSelector
    bind:value={item.client as string}
    items={$globalStore.clients}
    {errors}
    fieldName="client"
    uiName="Client"
  >
    {#snippet optionTemplate({ name })}{name}{/snippet}
  </DsSelector> -->

  {#if item.client !== "" && contacts.length > 0}
    <DsSelector
      bind:value={item.contact as string}
      items={contacts}
      {errors}
      fieldName="contact"
      uiName="Contact"
      clear={true}
    >
      {#snippet optionTemplate(item: ContactsResponse)}
        {item.surname}, {item.given_name}
      {/snippet}
    </DsSelector>
  {/if}

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
      {#each newCategories as categoryName}
        <span class="flex items-center rounded-full bg-neutral-200 px-2">
          <span>{categoryName}</span>
          <button
            class="text-neutral-500"
            onclick={preventDefault(
              () => (newCategories = newCategories.filter((name) => name !== categoryName)),
            )}
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
