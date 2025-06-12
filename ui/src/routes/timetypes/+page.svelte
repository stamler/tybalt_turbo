<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsTokenInput from "$lib/components/DSTokenInput.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { pb } from "$lib/pocketbase";
  import type { TimeTypesResponse } from "$lib/pocketbase-types";
  import { timeTypes } from "$lib/stores/time_types";

  timeTypes.init();

  let errors = $state({} as any);
  const defaultItem = {
    code: "",
    name: "",
    description: "",
    allowed_fields: [] as string[],
    required_fields: [] as string[],
  };

  let item = $state({ ...defaultItem });

  async function save() {
    try {
      await pb.collection("time_types").create(item);

      // save was successful, clear the form and refresh the divisions
      clearForm();
    } catch (error: any) {
      // if error.data.data is not an empty object, then set errors to that,
      // otherwise set errors to error.data.message
      if (error.data.data !== undefined && Object.keys(error.data.data).length > 0) {
        errors = error.data.data;
      } else {
        errors = { global: { message: error.data.message } };
      }
    }
  }

  function clearForm() {
    item = { ...defaultItem };
    errors = {};
  }
</script>

{#snippet anchor({ code }: TimeTypesResponse)}{code}{/snippet}
{#snippet headline({ name }: TimeTypesResponse)}{name}{/snippet}
{#snippet line1({ description }: TimeTypesResponse)}{description}{/snippet}
{#snippet line2({ allowed_fields }: TimeTypesResponse)}
  {#if allowed_fields !== null}
    <span class="opacity-30">allowed</span>
    {allowed_fields.join(", ")}
  {/if}
{/snippet}
{#snippet line3({ required_fields }: TimeTypesResponse)}
  {#if required_fields !== null}
    <span class="opacity-30">required</span>
    {required_fields.join(", ")}
  {/if}
{/snippet}

<!-- Show the list of items here -->
<DsList
  items={$timeTypes.items as TimeTypesResponse[]}
  inListHeader="Time Types"
  search={true}
  {anchor}
  {headline}
  {line1}
  {line2}
  {line3}
/>

<!-- Create a new job -->
<form class="flex w-full flex-col items-center gap-2 p-2">
  <DsTextInput bind:value={item.code as string} {errors} fieldName="code" uiName="Code" />
  <DsTextInput bind:value={item.name as string} {errors} fieldName="name" uiName="Name" />
  <DsTextInput
    bind:value={item.description as string}
    {errors}
    fieldName="description"
    uiName="Description"
  />
  <DsTokenInput
    bind:value={item.allowed_fields}
    {errors}
    fieldName="allowed_fields"
    uiName="Allowed Fields"
  />
  <DsTokenInput
    bind:value={item.required_fields}
    {errors}
    fieldName="required_fields"
    uiName="Required Fields"
  />
  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton action={save}>Save</DsActionButton>
      <DsActionButton action={clearForm}>Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
