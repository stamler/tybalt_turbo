<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import type { PageData } from "./$types";
  import { pb } from "$lib/pocketbase";
  import type { TimeTypesRecord } from "$lib/pocketbase-types";

  let { data }: { data: PageData } = $props();

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
      const record = await pb.collection("time_types").create(item, { returnRecord: true });
      if (data.timetypes === undefined) throw new Error("data.timetypes is undefined");
      data.timetypes.push(record);

      // submission was successful, clear the errors
      errors = {};

      // clear the item
      item = { ...defaultItem };
    } catch (error: any) {
      errors = error.data.data;
    }
  }
</script>

{#snippet anchor({ code })}{code}{/snippet}
{#snippet headline({ name })}{name}{/snippet}
{#snippet line3({ description })}{description}{/snippet}

<!-- Show the list of items here -->
<DsList items={data.timetypes as TimeTypesRecord[]} {anchor} {headline} {line3} />

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
