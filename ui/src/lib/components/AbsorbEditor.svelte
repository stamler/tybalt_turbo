<script lang="ts">
  import type { ClientsResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";
  import DsSelector from "./DSSelector.svelte";
  import DsActionButton from "./DSActionButton.svelte";
  import { pb } from "$lib/pocketbase";

  let {
    collectionName,
    targetRecordId,
    targetRecordSnippet,
  }: {
    collectionName: string;
    targetRecordId: string;
    targetRecordSnippet: any;
  } = $props();

  let errors = $state<Record<string, { message: string }>>({});
  let recordsToAbsorb = $state<string[]>([]);
  let selectedRecord = $state<string>("");

  $effect(() => {
    if (selectedRecord && !recordsToAbsorb.includes(selectedRecord)) {
      recordsToAbsorb = [...recordsToAbsorb, selectedRecord];
      selectedRecord = ""; // Reset selection after adding
    }
  });

  async function handleAbsorb() {
    try {
      await pb.send(`/api/${collectionName}/${targetRecordId}/absorb`, {
        method: "POST",
        body: {
          ids_to_absorb: recordsToAbsorb,
        },
      });

      // Redirect back to the list view after successful absorption
      window.location.href = `/${collectionName}/list`;
    } catch (error: unknown) {
      if (error instanceof Error) {
        errors = { global: { message: error.message } };
      } else {
        errors = { global: { message: "An unknown error occurred" } };
      }
    }
  }

  function handleCancel() {
    window.location.href = `/${collectionName}/list`;
  }

  function removeRecord(id: string) {
    recordsToAbsorb = recordsToAbsorb.filter((recordId) => recordId !== id);
  }
</script>

<div class="flex w-full flex-col gap-4 p-4">
  <div class="flex flex-col gap-2">
    <h2 class="text-xl font-bold">Target Record</h2>
    {@render targetRecordSnippet()}
  </div>

  <div class="flex flex-col gap-2">
    <h2 class="text-xl font-bold">Records to Absorb</h2>
    <DsSelector
      bind:value={selectedRecord}
      items={$globalStore.clients.filter(
        (c) => c.id !== targetRecordId && !recordsToAbsorb.includes(c.id),
      )}
      {errors}
      fieldName="records_to_absorb"
      uiName="Select Record"
    >
      {#snippet optionTemplate(item: ClientsResponse)}
        {item.name}
      {/snippet}
    </DsSelector>

    {#if recordsToAbsorb.length > 0}
      <ul class="flex flex-col gap-2">
        {#each recordsToAbsorb as recordId}
          {#each $globalStore.clients.filter((c: ClientsResponse) => c.id === recordId) as record}
            <li class="flex items-center gap-2 rounded bg-neutral-100 p-2">
              <span>{record.name}</span>
              <DsActionButton
                action={() => removeRecord(record.id)}
                icon="mdi:delete"
                title="Remove"
                color="red"
              />
            </li>
          {/each}
        {/each}
      </ul>
    {/if}
  </div>

  <div class="flex gap-2">
    <DsActionButton action={handleAbsorb}>Absorb</DsActionButton>
    <DsActionButton action={handleCancel}>Cancel</DsActionButton>
  </div>

  {#if errors.global}
    <div class="text-red-600">{errors.global.message}</div>
  {/if}
</div>
