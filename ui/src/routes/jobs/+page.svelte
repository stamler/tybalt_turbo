<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import { pb } from "$lib/pocketbase";
  import type { JobsRecord } from "$lib/pocketbase-types";

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
      if (data.jobs === undefined) throw new Error("data.jobs is undefined");
      data.jobs.push(record);

      // submission was successful, clear the errors
      errors = {};

      // clear the item
      item = { ...defaultItem };
    } catch (error: any) {
      errors = error.data.data;
    }
  }
</script>

<h1 class="text-green-800">Jobs</h1>

{#snippet anchor({ number })}{number}{/snippet}
{#snippet headline({ description })}{description}{/snippet}

{#snippet actions({ id })}
  <a href="/details/{id}">details</a>
  <a href="/{id}">delete</a>
{/snippet}

<!-- Show the list of items here -->
<DsList items={data.jobs as JobsRecord[]} {anchor} {headline} {actions} />

<!-- Create a new job -->
<form class="flex flex-col items-center w-full gap-2 p-2">
  <div class="flex flex-col w-full gap-2 {errors.number !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <label for="number">Job Number</label>
      <input
        class="flex-1"
        type="text"
        name="number"
        placeholder="Job Number"
        bind:value={item.number}
      />
    </span>
    {#if errors.number !== undefined}
      <span class="text-red-600">{errors.number.message}</span>
    {/if}
  </div>
  <div class="flex flex-col w-full gap-2 {errors.number !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <label for="description">Description</label>
      <input
        class="flex-1"
        type="text"
        name="description"
        placeholder="Job Description"
        bind:value={item.description}
      />
    </span>
    {#if errors.description !== undefined}
      <span class="text-red-600">{errors.description.message}</span>
    {/if}
  </div>
  <div class="flex flex-col w-full gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <button type="button" onclick={save}> Save </button>
      <button type="button"> Cancel </button>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
