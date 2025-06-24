<script lang="ts">
  import DsActionButton from "./DSActionButton.svelte";
  import { fade } from "svelte/transition";
  import { pb } from "$lib/pocketbase";
  import { managers } from "$lib/stores/managers";
  import { globalStore } from "$lib/stores/global";

  type Reviewer = {
    id: string;
    time_sheet: string;
    reviewer: string;
    reviewed: string;
    surname: string;
    given_name: string;
  };

  // initialize the stores, noop if already initialized
  managers.init();

  let { collectionName }: { collectionName: string } = $props();

  let show = $state(false);
  let itemId = $state("");
  let newViewer = $state("");
  let reviewers = $state([] as Reviewer[]);
  let ownerName = $state("Share");
  let timesheetDate = $state("");

  function closeModal() {
    show = false;
    itemId = "";
    newViewer = "";
    reviewers = [];
    ownerName = "Share";
    timesheetDate = "";
  }

  async function reloadReviewers() {
    try {
      const summary: any = await pb.send("/api/time_sheets/" + itemId + "/reviewers", {
        method: "GET",
      });

      reviewers = summary.reviewers || [];
      ownerName = summary.owner_name || "Share";

      if (summary.week_ending) {
        const dateObj = new Date(summary.week_ending);
        timesheetDate = dateObj.toLocaleDateString(undefined, {
          year: "numeric",
          month: "short",
          day: "numeric",
        });
      } else {
        timesheetDate = "";
      }

      newViewer = "";
    } catch (error: any) {
      globalStore.addError(`Failed to load reviewers: ${error}`);
      reviewers = []; // Ensure we have valid state even on error
    }
  }

  export async function openModal(id: string) {
    show = true;
    itemId = id;
    reloadReviewers();
  }

  async function addViewer() {
    if (!newViewer) {
      globalStore.addError("Please select a manager to add as viewer");
      return;
    }

    try {
      await pb.collection(collectionName).create({
        time_sheet: itemId,
        reviewer: newViewer,
      });
      reloadReviewers();
    } catch (error: any) {
      globalStore.addError(`Failed to add viewer: ${error}`);
    }
  }

  async function deleteViewer(reviewerRecordId: string) {
    try {
      await pb.collection(collectionName).delete(reviewerRecordId);
      reloadReviewers();
    } catch (error: any) {
      globalStore.addError(`Failed to remove viewer: ${error}`);
    }
  }
</script>

{#if show}
  <div
    class="fixed inset-0 z-50 overflow-y-auto overflow-x-hidden"
    transition:fade={{ duration: 200 }}
  >
    <div class="fixed inset-0 z-10 bg-black bg-opacity-80"></div>
    <div
      class="relative z-20 mx-auto my-20 flex w-fit max-w-full flex-col rounded-lg bg-neutral-800 p-4 text-neutral-300"
    >
      <div class="flex items-baseline justify-between gap-2">
        <h1 class="text-xl font-bold">{ownerName}</h1>
        {#if timesheetDate}
          <h5 class="text-sm text-neutral-400">{timesheetDate}</h5>
        {/if}
      </div>
      <div class="my-2 flex flex-col items-stretch gap-2 overflow-auto">
        <div class="rounded bg-neutral-700 p-4">
          <h3 class="text-lg font-semibold">{reviewers.length === 0 ? "No " : ""}Viewers</h3>
          <!-- Grid layout: two columns (names | delete button) with vertical spacing -->
          <div class="grid grid-cols-[1fr_auto] items-center gap-x-2 gap-y-2">
            {#each reviewers as reviewer (reviewer.id)}
              <span>{reviewer.surname}, {reviewer.given_name}</span>
              <DsActionButton
                action={() => deleteViewer(reviewer.id)}
                icon="mdi:delete"
                color="red"
                title="Remove Viewer"
              />
            {/each}
          </div>
        </div>

        <span class="flex items-center gap-1">
          <select name="manager" bind:value={newViewer} class="rounded bg-neutral-700 p-1">
            <option disabled selected>- select manager -</option>
            {#each $managers.items as m (m.id)}
              {#if !reviewers.find((r) => r.reviewer === m.id)}
                <option value={m.id}>
                  {m.surname}, {m.given_name}
                </option>
              {/if}
            {/each}
          </select>
          <DsActionButton
            action={addViewer}
            icon="feather:plus-circle"
            color="green"
            title="Add Viewer"
          />
        </span>
      </div>
      <div class="gap-2">
        <DsActionButton action={closeModal}>Close</DsActionButton>
      </div>
    </div>
  </div>
{/if}
