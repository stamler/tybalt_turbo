<script lang="ts" generics="T extends BaseSystemFields<any>">
  import type {
    ClientsResponse,
    ClientContactsResponse,
    AbsorbActionsResponse,
  } from "$lib/pocketbase-types";
  import DsActionButton from "./DSActionButton.svelte";
  import DsAutoComplete from "./DSAutoComplete.svelte";
  import DsSelector from "./DSSelector.svelte";
  import { pb } from "$lib/pocketbase";
  import { page } from "$app/stores";
  import type { Snippet } from "svelte";
  import type { BaseSystemFields } from "$lib/pocketbase-types";
  import type MiniSearch from "minisearch";

  let {
    collectionName,
    targetRecordId,
    recordSnippet,
    availableRecords,
    autoCompleteIndex = null,
  }: {
    collectionName: string;
    targetRecordId: string;
    recordSnippet: Snippet<[T]>;
    availableRecords: (ClientsResponse | ClientContactsResponse)[];
    autoCompleteIndex?: MiniSearch<ClientsResponse | ClientContactsResponse> | null;
  } = $props();

  let errors = $state<Record<string, { message: string }>>({});
  let recordsToAbsorb = $state<string[]>([]);
  let selectedRecord = $state<string>("");
  let existingAbsorbAction = $state<AbsorbActionsResponse | null>(null);
  let showUndoConfirm = $state(false);
  let showCommitConfirm = $state(false);
  let items = $state<(ClientsResponse | ClientContactsResponse)[]>([]);
  let targetRecord = $state<T | null>(null);

  // Reference to the DsAutoComplete component so we can call its exposed focus() helper.
  // svelte-ignore non_reactive_update
  let autoCompleteRef: any = null;

  // Update items whenever availableRecords or recordsToAbsorb changes
  $effect(() => {
    items = availableRecords.filter(
      (c) => c.id !== targetRecordId && !recordsToAbsorb.includes(c.id),
    );
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
    // Don't add the target record, a blank record, or a record that is already
    // in the list, to the list of records to absorb.
    if (
      selectedRecord !== "" &&
      selectedRecord !== targetRecordId &&
      !recordsToAbsorb.includes(selectedRecord)
    ) {
      recordsToAbsorb = [...recordsToAbsorb, selectedRecord];
    }
    selectedRecord = ""; // Reset selection even if the record is already in the list
    // Re-focus the autocomplete's internal input so the user can keep typing.
    autoCompleteRef?.focus?.();
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

  function addRecord(id: string | number) {
    const idStr = id.toString();
    if (idStr !== targetRecordId && !recordsToAbsorb.includes(idStr)) {
      recordsToAbsorb = [...recordsToAbsorb, idStr];
    }
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
        {#if autoCompleteIndex}
          <DsAutoComplete
            multi
            bind:this={autoCompleteRef}
            index={autoCompleteIndex as unknown as MiniSearch<unknown>}
            excludeIds={[targetRecordId, ...recordsToAbsorb]}
            value={selectedRecord}
            choose={addRecord}
            {errors}
            fieldName="records_to_absorb"
            uiName="Select Record"
          >
            {#snippet resultTemplate(item: any)}
              {@render recordSnippet(item as unknown as T)}
            {/snippet}
          </DsAutoComplete>
        {:else}
          <DsSelector
            bind:value={selectedRecord}
            {items}
            {errors}
            fieldName="records_to_absorb"
            uiName="Select Record"
          >
            {#snippet optionTemplate(item: any)}
              {@render recordSnippet(item as unknown as T)}
            {/snippet}
          </DsSelector>
        {/if}
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
