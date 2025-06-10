<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import { pb } from "$lib/pocketbase";
  import { divisions } from "$lib/stores/divisions";
  import DsActionButton from "$lib/components/DSActionButton.svelte";

  // initialize the store, noop if already initialized
  divisions.init();

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

      // save was successful, clear the form
      clearForm();
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
<DsList items={$divisions.items} inListHeader="Divisions" search={true}>
  {#snippet anchor({ code })}{code}{/snippet}
  {#snippet headline({ name })}{name}{/snippet}
</DsList>

<!-- Create a new job -->
<form class="flex w-full flex-col items-center gap-2 p-2">
  <DsTextInput bind:value={item.code as string} {errors} fieldName="code" uiName="Code" />
  <DsTextInput bind:value={item.name as string} {errors} fieldName="name" uiName="Name" />
  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton action={save}>Save</DsActionButton>
      <DsActionButton action={clearForm}>Clear</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
