<script lang="ts">
  import Icon from "@iconify/svelte";
  import { fade } from "svelte/transition";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";

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

  async function rejectTimeSheet() {
    try {
      await pb.send("/api/reject-timesheet", {
        method: "POST",
        body: JSON.stringify({ timeSheetId: itemId, rejectionReason }),
        headers: {
          "Content-Type": "application/json",
        },
      });

      globalStore.refresh("time_sheets");
      closeModal();
    } catch (error) {
      console.error("Error rejecting timesheet:", error);
    }
  }
</script>

{#if show}
  <div
    class="z-90 fixed inset-0 overflow-y-auto overflow-x-hidden"
    transition:fade={{ duration: 200 }}
  >
    <div class="fixed inset-0 z-10 bg-black bg-opacity-80"></div>
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
          class="rounded bg-neutral-700 p-2"
        ></textarea>
      </div>
      <div class="px-2 pb-2 pt-1">
        <button onclick={rejectTimeSheet}>Reject</button>
        <button onclick={closeModal}>Cancel</button>
      </div>
    </div>
  </div>
{/if}
