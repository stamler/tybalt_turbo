<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import { pb } from "$lib/pocketbase";
  import type { TimeTypesRecord } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";

  let errors = $state({} as any);
  const defaultItem = {
    code: "",
    name: "",
    description: "",
    allowed_fields: [] as string[],
  };

  let item = $state({ ...defaultItem });

  async function save() {
    try {
      await pb.collection("time_types").create(item);

      // save was successful, clear the form and refresh the divisions
      clearForm();
      globalStore.refresh("time_types");
    } catch (error: any) {
      errors = error.data.data;
    }
  }

  function clearForm() {
    item = { ...defaultItem };
    errors = {};
  }
</script>

{#snippet anchor({ code })}{code}{/snippet}
{#snippet headline({ name })}{name}{/snippet}
{#snippet line1({ description })}{description}{/snippet}
{#snippet line2({ allowed_fields })}
  <span class="opacity-30">allowed</span>
  {allowed_fields.join(", ")}
{/snippet}
{#snippet line3({ required_fields })}
  <span class="opacity-30">required</span>
  {required_fields.join(", ")}
{/snippet}

<!-- Show the list of items here -->
<DsList
  items={$globalStore.time_types as TimeTypesRecord[]}
  search={true}
  {anchor}
  {headline}
  {line1}
  {line2}
  {line3}
/>

<!-- Create a new job -->
<form class="flex w-full flex-col items-center gap-2 p-2">
  <DsTextInput bind:value={item.code} {errors} fieldName="code" uiName="Code" />
  <DsTextInput bind:value={item.name} {errors} fieldName="name" uiName="Name" />
  <DsTextInput
    bind:value={item.description}
    {errors}
    fieldName="description"
    uiName="Description"
  />
  <!-- todo: add allowed_fields editor DsMultiStringInput-->
  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <button
        type="button"
        onclick={save}
        class="rounded-sm bg-yellow-200 px-1 hover:bg-yellow-300"
      >
        Save
      </button>
      <button type="button"> Cancel </button>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
