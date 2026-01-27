<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { goto } from "$app/navigation";
  import type { PageData } from "./$types";

  let { data }: { data: PageData } = $props();

  const revising = data.revising;
  const sourceSheet = data.sourceSheet;
  const nextRevision = data.nextRevision;

  let errors = $state({} as Record<string, { message: string }>);
  let item = $state({
    name: revising && sourceSheet ? sourceSheet.name : "",
    effective_date: "",
    revision: nextRevision,
    active: false,
  });

  async function save(event: Event) {
    event.preventDefault();

    try {
      const created = await pb.collection("rate_sheets").create(item);
      errors = {};
      goto(`/rate-sheets/${created.id}/details`);
    } catch (error: any) {
      errors = error.data?.data ?? {};
    }
  }
</script>

<form
  class="flex w-full flex-col items-center gap-2 p-2"
  enctype="multipart/form-data"
  onsubmit={save}
>
  <h1 class="text-xl font-bold">
    {revising ? "Revise Rate Sheet" : "New Rate Sheet"}
  </h1>

  {#if revising}
    <!-- Name field: pre-populated and disabled when revising -->
    <div class="flex w-full flex-col gap-2">
      <span class="flex w-full items-center gap-2">
        <label for="name">Name</label>
        <input
          class="flex-1 cursor-not-allowed rounded border border-neutral-300 bg-neutral-100 px-1 opacity-60"
          type="text"
          id="name"
          name="name"
          value={item.name}
          disabled
        />
        <span class="text-sm italic text-neutral-500">revising</span>
      </span>
    </div>
  {:else}
    <DsTextInput bind:value={item.name} {errors} fieldName="name" uiName="Name" />
  {/if}

  <div class="flex w-full flex-col gap-2 {errors.effective_date !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <label for="effective_date">Effective Date</label>
      <input
        class="flex-1 rounded border border-neutral-300 px-1"
        type="date"
        id="effective_date"
        name="effective_date"
        bind:value={item.effective_date}
      />
    </span>
    {#if errors.effective_date !== undefined}
      <span class="text-red-600">{errors.effective_date.message}</span>
    {/if}
  </div>

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton type="submit">Save</DsActionButton>
      <DsActionButton action="/rate-sheets/list">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
