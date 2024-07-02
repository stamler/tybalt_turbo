<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import { pb } from "$lib/pocketbase";
  import type { BaseSystemFields, DivisionsRecord } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";

  let errors = $state({} as any);
  const defaultItem = {
    code: "",
    name: "",
    allowed_fields: [] as string[],
  };

  let item = $state({ ...defaultItem });

  async function save() {
    try {
      await pb.collection("divisions").create(item);

      // save was successful, clear the form and refresh the divisions
      clearForm();
      globalStore.refresh("divisions");
    } catch (error: any) {
      errors = error.data.data;
    }
  }

  function clearForm() {
    item = { ...defaultItem };
    errors = {};
  }
</script>

<!-- Show the list of items here -->
<DsList items={$globalStore.divisions as (DivisionsRecord & BaseSystemFields)[]} search={true}>
  {#snippet anchor({ code })}{code}{/snippet}
  {#snippet headline({ name })}{name}{/snippet}
</DsList>

<!-- Create a new job -->
<form class="flex w-full flex-col items-center gap-2 p-2">
  <DsTextInput bind:value={item.code} {errors} fieldName="code" uiName="Code" />
  <DsTextInput bind:value={item.name} {errors} fieldName="name" uiName="Name" />
  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <button
        type="button"
        onclick={save}
        class="rounded-sm bg-yellow-200 px-1 hover:bg-yellow-300"
      >
        Save
      </button>
      <button type="button" onclick={clearForm}> Cancel </button>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
