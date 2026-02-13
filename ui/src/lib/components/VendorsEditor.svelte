<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsActionButton from "./DSActionButton.svelte";
  import { goto } from "$app/navigation";
  import type { VendorsPageData } from "$lib/svelte-types";
  import { untrack } from "svelte";
  import { expensesEditingEnabled } from "$lib/stores/appConfig";
  import DsEditingDisabledBanner from "./DsEditingDisabledBanner.svelte";

  let { data }: { data: VendorsPageData } = $props();

  let errors = $state({} as any);
  let item = $state(untrack(() => data.item));

  async function save(event: Event) {
    event.preventDefault();

    try {
      if (data?.editing && data?.id !== null) {
        await pb.collection("vendors").update(data.id, item);
      } else {
        await pb.collection("vendors").create(item);
      }
      errors = {};
      goto("/vendors/list");
    } catch (error: any) {
      errors = error.data.data;
    }
  }
</script>

{#if !$expensesEditingEnabled}
  <DsEditingDisabledBanner
    message="Vendor editing is currently disabled during a system transition."
  />
{/if}

<form
  class="flex w-full flex-col items-center gap-2 p-2 max-lg:[&_button]:text-base max-lg:[&_input]:text-base max-lg:[&_label]:text-base max-lg:[&_select]:text-base max-lg:[&_textarea]:text-base"
  enctype="multipart/form-data"
  onsubmit={save}
>
  <h1 class="w-full text-xl font-bold text-neutral-800">
    {#if data.editing}
      {#if item.name}
        Editing {item.name}
      {:else}
        Editing Vendor
      {/if}
    {:else}
      Create Vendor
    {/if}
  </h1>

  <DsTextInput bind:value={item.name as string} {errors} fieldName="name" uiName="Name" />

  <DsTextInput bind:value={item.alias as string} {errors} fieldName="alias" uiName="Alias" />

  <DsSelector
    bind:value={item.status as string}
    items={[
      { id: "Active", name: "Active" },
      { id: "Inactive", name: "Inactive" },
    ]}
    {errors}
    fieldName="status"
    uiName="Status"
  >
    {#snippet optionTemplate(item)}
      {item.name}
    {/snippet}
  </DsSelector>

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton type="submit">Save</DsActionButton>
      <DsActionButton action="/vendors/list">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
