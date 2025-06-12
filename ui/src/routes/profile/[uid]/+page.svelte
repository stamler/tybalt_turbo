<script lang="ts">
  import type { PageData } from "./$types";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { globalStore } from "$lib/stores/global";
  import type { ProfilesResponse } from "$lib/pocketbase-types";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import { authStore } from "$lib/stores/auth";
  import DSAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { divisions } from "$lib/stores/divisions";
  import { managers } from "$lib/stores/managers";

  // initialize the stores, noop if already initialized
  divisions.init();
  managers.init();

  let { data }: { data: PageData } = $props();
  let errors = $state({} as any);
  let item = $state(data.item as ProfilesResponse);

  // Sync item with data.item when it changes (e.g., when navigating to different profiles)
  $effect(() => {
    item = data.item as ProfilesResponse;
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

      // TODO: notify the user that save was successful
    } catch (error: any) {
      errors = error.data.data;
    }
  }
</script>

<form class="flex w-full flex-col gap-2 p-2">
  <DsTextInput
    bind:value={item.given_name as string}
    {errors}
    fieldName="given_name"
    uiName="Given Name"
  />
  <DsTextInput bind:value={item.surname as string} {errors} fieldName="surname" uiName="Surname" />
  {#if $managers.index !== null}
    <DSAutoComplete
      bind:value={item.manager as string}
      index={$managers.index}
      {errors}
      fieldName="manager"
      uiName="Manager"
    >
      {#snippet resultTemplate(item)}{item.surname}, {item.given_name}{/snippet}
    </DSAutoComplete>
    <DSAutoComplete
      bind:value={item.alternate_manager as string}
      index={$managers.index}
      {errors}
      fieldName="alternate_manager"
      uiName="Alternate Manager"
    >
      {#snippet resultTemplate(item)}{item.surname}, {item.given_name}{/snippet}
    </DSAutoComplete>
  {/if}
  {#if $divisions.index !== null}
    <DSAutoComplete
      bind:value={item.default_division as string}
      index={$divisions.index}
      {errors}
      fieldName="default_division"
      uiName="Default Division"
    >
      {#snippet resultTemplate(item)}{item.code} - {item.name}{/snippet}
    </DSAutoComplete>
  {/if}
  <p>Token expiration date: {authStore.tokenExpirationDate() ?? "No token"}</p>
  <span class="flex w-full gap-2">
    <DsActionButton action={save}>Save</DsActionButton>
  </span>

  <p>Claims:</p>
  <ul class="flex flex-row gap-2">
    {#each $globalStore.claims as claim}
      <DsLabel color="cyan">{claim}</DsLabel>
    {/each}
  </ul>

  <p>User PO Permission Data:</p>
  <p>{JSON.stringify($globalStore.user_po_permission_data)}</p>
</form>
