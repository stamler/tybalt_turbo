<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import { pb } from "$lib/pocketbase";
  import type { JobsResponse } from "$lib/pocketbase-types";
  import DsTextInput from "$lib/components/DSTextInput.svelte";

  let { data }: { data: PageData } = $props();

  let errors = $state({} as any);
  const defaultItem = {
    number: "",
    description: "",
  };

  let item = $state({ ...defaultItem });

  async function save() {
    try {
      const record = await pb.collection("jobs").create(item, { returnRecord: true });
      if (data.items === undefined) throw new Error("data.items is undefined");
      data.items.push(record);

      // TODO:
      // 1. don't use page load function for jobs, get from index instead
      // 2. find a way to show later items from the index
      // 3. add the new item to the index on save

      // submission was successful, clear the errors
      errors = {};

      // clear the item
      item = { ...defaultItem };
    } catch (error: any) {
      errors = error.data.data;
    }
  }
  async function del(id: string): Promise<void> {
    // return immediately if data.items is not an array
    if (!Array.isArray(data.items)) return;

    try {
      await pb.collection("jobs").delete(id);

      // remove the item from the list
      data.items = data.items.filter((item) => item.id !== id);
    } catch (error: any) {
      alert(error.data.message);
    }
  }
</script>

{#snippet anchor({ number }: JobsResponse)}{number}{/snippet}
{#snippet headline({ description }: JobsResponse)}{description}{/snippet}

{#snippet actions({ id }: JobsResponse)}
  <a href="/details/{id}">details</a>
  <button type="button" onclick={() => del(id)}>delete</button>
{/snippet}

<!-- Show the list of items here -->
<DsList items={data.items as JobsResponse[]} search={true} {anchor} {headline} {actions} />

<!-- Create a new job -->
<form class="flex w-full flex-col items-center gap-2 p-2">
  <DsTextInput bind:value={item.number} {errors} fieldName="number" uiName="Job Number" />
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
