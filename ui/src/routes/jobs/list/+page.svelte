<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import { pb } from "$lib/pocketbase";
  import type { JobsResponse } from "$lib/pocketbase-types";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  let { data }: { data: PageData } = $props();

  let errors = $state({} as any);
  const defaultItem = {
    number: "",
    description: "",
  };

  let item = $state({ ...defaultItem });
  let items = $state(data.items);

  async function save() {
    try {
      const record = await pb.collection("jobs").create(item, { returnRecord: true });
      if (items === undefined) throw new Error("items is undefined");
      items.push(record);

      // TODO:
      // 1. don't use page load function for jobs, get from index instead
      // 2. find a way to show later items from the index
      // 3. add the new item to the index on save

      // submission was successful, clear the errors
      errors = {};

      // clear the item
      clearForm();
    } catch (error: any) {
      errors = error.data.data;
    }
  }
  async function del(id: string): Promise<void> {
    // return immediately if items is not an array
    if (!Array.isArray(items)) return;

    try {
      await pb.collection("jobs").delete(id);

      // remove the item from the list
      items = items.filter((item) => item.id !== id);
    } catch (error: any) {
      alert(error.data.message);
    }
  }

  function clearForm() {
    item = { ...defaultItem };
    errors = {};
  }
</script>

{#snippet anchor({ number }: JobsResponse)}{number}{/snippet}
{#snippet headline({ description }: JobsResponse)}{description}{/snippet}
{#snippet line1({ expand }: JobsResponse)}
  <span class="flex gap-1">
    {#each expand?.categories_via_job as category}
      <span class="rounded-sm border border-cyan-100 bg-cyan-50 px-1">{category.name}</span>
    {/each}
  </span>
{/snippet}
{#snippet actions({ id }: JobsResponse)}
  <DsActionButton action="/jobs/{id}/edit" icon="mdi:pencil" title="Edit" color="blue" />
  <DsActionButton action="/details/{id}" icon="mdi:more-circle" title="More Details" color="blue" />
  <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
{/snippet}

<!-- Show the list of items here -->
<DsList items={items as JobsResponse[]} search={true} {anchor} {headline} {line1} {actions} />
