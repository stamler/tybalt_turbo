<script lang="ts" generics="T extends BaseSystemFields<any>">
  import type { AbsorbActionsResponse } from "$lib/pocketbase-types";
  import DsActionButton from "./DSActionButton.svelte";
  import DsAutoComplete from "./DSAutoComplete.svelte";
  import DsSelector from "./DSSelector.svelte";
  import { pb } from "$lib/pocketbase";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import type { Snippet } from "svelte";
  import type { BaseSystemFields } from "$lib/pocketbase-types";
  import type MiniSearch from "minisearch";
  import AbsorbList from "./AbsorbList.svelte";
  import { getAbsorbRedirectUrl } from "$lib/utilities";
  import { appConfig, jobsEditingEnabled } from "$lib/stores/appConfig";

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
    availableRecords: T[];
    autoCompleteIndex?: MiniSearch<T> | null;
  } = $props();

  // Initialize config store
  appConfig.init();

  // Collections that affect jobs when absorbed
  const collectionsAffectingJobs = ["clients", "client_contacts"];

  // Derived: absorb is disabled if this collection affects jobs and job editing is disabled
  const absorbDisabled = $derived(
    collectionsAffectingJobs.includes(collectionName) && !$jobsEditingEnabled,
  );

  let errors = $state<Record<string, { message: string }>>({});
  let recordsToAbsorb = $state<string[]>([]);
  let selectedRecord = $state<string>("");
  let existingAbsorbAction = $state<AbsorbActionsResponse | null>(null);
  let items = $state<T[]>([]);
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

  function goBack() {
    if (typeof window !== "undefined" && window.history.length > 1) {
      window.history.back();
    } else {
      getAbsorbRedirectUrl(collectionName, targetRecordId, $page.params.cid).then(goto);
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

  // Undo/commit handling is managed by AbsorbList

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

      // Redirect to absorb actions page after successful absorption
      goto("/absorb/actions");
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
  {#if absorbDisabled}
    <div class="disabled-notice">
      <p>
        Record absorption is temporarily disabled for <strong>{collectionName}</strong> during a system
        transition.
      </p>
      <p>Absorbing these records would modify job data which is currently locked.</p>
      <div class="mt-4">
        <DsActionButton action={goBack} color="neutral">Go Back</DsActionButton>
      </div>
    </div>
  {:else if existingAbsorbAction}
    <div class="rounded border-2 border-yellow-500 bg-yellow-50 p-4">
      <div class="mb-2 text-yellow-900">
        <p class="font-semibold">There is a pending absorb action for this collection.</p>
        <p class="text-sm">
          You must either undo the previous absorption or commit it before performing a new absorb
          operation.
        </p>
      </div>
      <div class="mb-3">
        <DsActionButton action={goBack} color="neutral">Back</DsActionButton>
      </div>
      <AbsorbList {collectionName} />
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
      <DsActionButton
        action={() =>
          getAbsorbRedirectUrl(collectionName, targetRecordId, $page.params.cid).then(goto)}
        >Cancel</DsActionButton
      >
    </div>

    {#if errors.global}
      <div class="text-red-600">{errors.global.message}</div>
    {/if}
  {/if}
</div>

<style>
  .disabled-notice {
    padding: 1.5rem;
    background-color: #fff3cd;
    border: 1px solid #ffc107;
    border-radius: 0.5rem;
    max-width: 600px;
  }

  .disabled-notice p {
    margin: 0.5rem 0;
    color: #856404;
  }
</style>
