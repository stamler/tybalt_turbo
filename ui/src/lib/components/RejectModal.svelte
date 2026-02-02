<script lang="ts">
  import DsActionButton from "./DSActionButton.svelte";
  import { fade } from "svelte/transition";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import type { CollectionName } from "$lib/stores/global";
  import { createEventDispatcher } from "svelte";

  const dispatch = createEventDispatcher();

  let { collectionName }: { collectionName: string } = $props();
  let show = $state(false);
  let itemId = $state("");
  let rejectionReason = $state("");

  function closeModal() {
    show = false;
    rejectionReason = "";
  }

  export function openModal(id: string) {
    show = true;
    itemId = id;
  }

  async function rejectRecord() {
    try {
      await pb.send(`/api/${collectionName}/${itemId}/reject`, {
        method: "POST",
        body: JSON.stringify({ rejection_reason: rejectionReason }),
        headers: {
          "Content-Type": "application/json",
        },
      });

      globalStore.refresh(collectionName as CollectionName);
      closeModal();

      // emit event to refresh the page to the parent
      dispatch("refresh");
    } catch (error) {
      globalStore.addError(error?.response?.message);
      closeModal();
    }
  }
</script>

{#if show}
  <div
    class="fixed inset-0 z-50 overflow-x-hidden overflow-y-auto"
    transition:fade={{ duration: 200 }}
  >
    <div class="bg-opacity-80 fixed inset-0 z-10 bg-black"></div>
    <div
      class="relative z-20 mx-auto my-20 flex w-11/12 flex-col rounded-lg bg-neutral-800 p-4 text-neutral-300 md:w-3/5"
    >
      <div class="flex items-start justify-between">
        <h1>Reject</h1>
        <h5>{itemId}</h5>
      </div>
      <div class="my-2 flex flex-col items-stretch gap-2 overflow-auto">
        <textarea
          bind:value={rejectionReason}
          placeholder="Enter rejection reason"
          class="rounded-sm bg-neutral-700 p-2"
        ></textarea>
      </div>
      <div class="gap-2">
        <DsActionButton action={rejectRecord}>Reject</DsActionButton>
        <DsActionButton action={closeModal}>Cancel</DsActionButton>
      </div>
    </div>
  </div>
{/if}
