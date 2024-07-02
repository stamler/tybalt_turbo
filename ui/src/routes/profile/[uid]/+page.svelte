<script lang="ts">
  import type { PageData } from "./$types";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import { globalStore } from "$lib/stores/global";
  import type { CollectionName } from "$lib/stores/global";

  let { data }: { data: PageData } = $props();
  let errors = $state({} as any);
  let item = data.item;

  const collectionAges = $derived.by(() => {
    // return an array of objects with the key and age of the corresponding
    // collection
    return (Object.keys($globalStore.collections) as CollectionName[]).map((key) => {
      const collection = $globalStore.collections[key];
      return {
        key,
        age: Math.round((new Date().getTime() - collection.lastRefresh.getTime()) / 1000),
      };
    });
  });

  async function save() {
    try {
      if (data.editing && data.id !== null) {
        // update the item
        await pb.collection("profiles").update(data.id, item);
      } else {
        // create a new item
        const record = await pb.collection("profiles").create(item, { returnRecord: true });
        // if the save was successful, the editing property needs to be set to
        // true if furthers saves are to be successful otherwise we'll have a
        // duplicate item error from the server
        data.id = record.id;
        data.editing = true;
      }

      // submission was successful, clear the errors
      errors = {};

      globalStore.refresh("managers");

      // TODO: notify the user that save was successful
    } catch (error: any) {
      errors = error.data.data;
    }
  }
</script>

<form class="flex w-full flex-col gap-2 p-2">
  <DsTextInput bind:value={item.given_name} {errors} fieldName="given_name" uiName="Given Name" />
  <DsTextInput bind:value={item.surname} {errors} fieldName="surname" uiName="Surname" />
  {#snippet managerOptionTemplate(item)}
    {item.surname}, {item.given_name}
  {/snippet}
  <DsSelector
    bind:value={item.manager}
    items={$globalStore.managers}
    {errors}
    optionTemplate={managerOptionTemplate}
    fieldName="manager"
    uiName="Manager"
  />
  <DsSelector
    bind:value={item.alternate_manager}
    items={$globalStore.managers}
    {errors}
    optionTemplate={managerOptionTemplate}
    fieldName="alternate_manager"
    uiName="Alternate Manager"
  />
  {#snippet divisionsOptionTemplate(item)}
    {item.code} - {item.name}
  {/snippet}
  <DsSelector
    bind:value={item.default_division}
    items={$globalStore.divisions}
    {errors}
    optionTemplate={divisionsOptionTemplate}
    fieldName="division"
    uiName="Default Division"
  />
  <span class="flex w-full gap-2">
    <button type="button" onclick={save} class="rounded-sm bg-yellow-200 px-1 hover:bg-yellow-300">
      Save
    </button>
  </span>
  <p>Store ages:</p>
  <ul>
    {#each collectionAges as age}
      <li>{age.key}: {age.age}s</li>
    {/each}
  </ul>
</form>
