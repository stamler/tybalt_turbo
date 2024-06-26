<script lang="ts">
  import type { PageData } from "./$types";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import { globalStore } from "$lib/stores/global";

  let { data }: { data: PageData } = $props();
  let errors = $state({} as any);
  let item = data.item;

  async function save() {
    try {
      if (data.editing && data.id !== null) {
        // update the item
        await pb.collection("profiles").update(data.id, item);
      } else {
        // create a new item
        await pb.collection("profiles").create(item);
      }

      // submission was successful, clear the errors
      errors = {};

      // TODO: notify the user that save was successful

    } catch (error: any) {
      errors = error.data.data;
    }
  }

</script>

<form class="flex flex-col items-center w-full gap-2 p-2">
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

  <button type="button" onclick={save} class="bg-yellow-200 rounded-sm px-1 hover:bg-yellow-300"> Save </button>

</form>
