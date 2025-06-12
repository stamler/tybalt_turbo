<script lang="ts" generics="T extends BaseSystemFields<any>">
  import type {
    ClientsResponse,
    ClientContactsResponse,
    AbsorbActionsResponse,
  } from "$lib/pocketbase-types";
  import { clients } from "$lib/stores/clients";
  import DsSelector from "./DSSelector.svelte";
  import DsActionButton from "./DSActionButton.svelte";
  import { pb } from "$lib/pocketbase";
  import { page } from "$app/stores";
  import type { Snippet } from "svelte";
  import type { BaseSystemFields } from "$lib/pocketbase-types";

  let {
    collectionName,
    targetRecordId,
    recordSnippet,
  }: {
    collectionName: string;
    targetRecordId: string;
    recordSnippet: Snippet<[T]>;
  } = $props();

  let errors = $state<Record<string, { message: string }>>({});
  let recordsToAbsorb = $state<string[]>([]);
  let selectedRecord = $state<string>("");
  let existingAbsorbAction = $state<AbsorbActionsResponse | null>(null);
  let showUndoConfirm = $state(false);
  let showCommitConfirm = $state(false);
  let availableRecords = $state<(ClientsResponse | ClientContactsResponse)[]>([]);
  let items = $state<(ClientsResponse | ClientContactsResponse)[]>([]);
  let targetRecord = $state<T | null>(null);

  // Update items whenever availableRecords or recordsToAbsorb changes
  $effect(() => {
    items = availableRecords.filter(
      (c) => c.id !== targetRecordId && !recordsToAbsorb.includes(c.id),
    );
  });

  // Get available records based on collection and context
  $effect(() => {
    if (collectionName === "client_contacts" && $page.params.cid) {
      // For client contacts, filter by the current client
      pb.collection("client_contacts")
        .getFullList<ClientContactsResponse>({ filter: `client = "${$page.params.cid}"` })
        .then((contacts) => {
          availableRecords = contacts;
        });
    } else if (collectionName === "clients") {
      availableRecords = $clients.items || [];
    }
  });

  // Check for existing absorb action on mount
  $effect(() => {
    checkExistingAbsorbAction();
  });

  // Get target record
  $effect(() => {
    const record = availableRecords.find((r) => r.id === targetRecordId);
    if (record) {
      targetRecord = record as unknown as T;
    }
  });

  function redirectBack() {
    if (collectionName === "client_contacts") {
      window.location.href = `/clients/${$page.params.cid}/edit`;
    } else {
      window.location.href = `/${collectionName}/list`;
    }
  }

  async function checkExistingAbsorbAction() {
    try {
      const result = await pb
        .collection("absorb_actions")
        .getFirstListItem(`collection_name="${collectionName}"`);
      existingAbsorbAction = result;
    } catch (error) {
      // No absorb action exists, which is fine
      existingAbsorbAction = null;
    }
  }

  async function handleUndo() {
    try {
      await pb.send(`/api/${collectionName}/undo_absorb`, {
        method: "POST",
      });
      redirectBack();
    } catch (error: unknown) {
      if (error instanceof Error) {
        errors = { global: { message: error.message } };
      } else {
        errors = { global: { message: "An unknown error occurred" } };
      }
    }
  }

  async function handleCommit() {
    if (existingAbsorbAction === null) return;
    try {
      await pb.collection("absorb_actions").delete(existingAbsorbAction.id);
      redirectBack();
    } catch (error: unknown) {
      if (error instanceof Error) {
        errors = { global: { message: error.message } };
      } else {
        errors = { global: { message: "An unknown error occurred" } };
      }
    }
  }

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
      redirectBack();
    } catch (error: unknown) {
      if (error instanceof Error) {
        errors = { global: { message: error.message } };
      } else {
        errors = { global: { message: "An unknown error occurred" } };
      }
    }
  }

  function removeRecord(id: string) {
    recordsToAbsorb = recordsToAbsorb.filter((recordId) => recordId !== id);
  }
</script>

<div class="flex w-full flex-col gap-4 p-4">
  {#if existingAbsorbAction}
    <div class="rounded-lg border-2 border-yellow-500 bg-yellow-50 p-4">
      <h2 class="mb-2 text-lg font-bold text-yellow-800">Pending Absorb Action</h2>
      <p class="mb-4 text-yellow-700">
        There is a pending absorb action for this collection. You must either undo the previous
        absorption or commit it before performing a new absorb operation.
      </p>
      <div class="flex gap-2">
        {#if showUndoConfirm}
          <div class="flex flex-col gap-2">
            <p class="font-bold text-red-600">
              Are you sure you want to undo the previous absorption? This action cannot be reversed.
            </p>
            <div class="flex gap-2">
              <DsActionButton action={handleUndo} color="red">Confirm Undo</DsActionButton>
              <DsActionButton action={() => (showUndoConfirm = false)}>Cancel</DsActionButton>
            </div>
          </div>
        {:else if showCommitConfirm}
          <div class="flex flex-col gap-2">
            <p class="font-bold text-red-600">
              Are you sure you want to commit the previous absorption? This action cannot be
              reversed.
            </p>
            <div class="flex gap-2">
              <DsActionButton action={handleCommit} color="red">Confirm Commit</DsActionButton>
              <DsActionButton action={() => (showCommitConfirm = false)}>Cancel</DsActionButton>
            </div>
          </div>
        {:else}
          <DsActionButton action={() => (showUndoConfirm = true)} color="yellow"
            >Undo Previous Absorb</DsActionButton
          >
          <DsActionButton action={() => (showCommitConfirm = true)} color="green"
            >Commit Previous Absorb</DsActionButton
          >
        {/if}
      </div>
    </div>
  {:else}
    <div class="flex flex-col gap-2">
      <h2 class="text-xl font-bold">Target Record</h2>
      {#if targetRecord}
        {@render recordSnippet(targetRecord)}
      {/if}
    </div>

    <div class="flex flex-col gap-2">
      <h2 class="text-xl font-bold">Records to Absorb</h2>
      {#if items.length > 0}
        <DsSelector
          bind:value={selectedRecord}
          {items}
          {errors}
          fieldName="records_to_absorb"
          uiName="Select Record"
        >
          {#snippet optionTemplate(item)}
            {@render recordSnippet(item as unknown as T)}
          {/snippet}
        </DsSelector>
      {:else}
        <p class="text-neutral-500">No more records available to absorb</p>
      {/if}

      {#if recordsToAbsorb.length > 0}
        <ul class="flex flex-col gap-2">
          {#each recordsToAbsorb as recordId}
            {#each availableRecords.filter((c) => c.id === recordId) as record}
              <li class="flex items-center gap-2 rounded bg-neutral-100 p-2">
                {@render recordSnippet(record as unknown as T)}
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
      <DsActionButton action={redirectBack}>Cancel</DsActionButton>
    </div>

    {#if errors.global}
      <div class="text-red-600">{errors.global.message}</div>
    {/if}
  {/if}
</div>
