<script lang="ts">
  import Icon from "@iconify/svelte";
  import DsActionButton from "./DSActionButton.svelte";
  import { fade } from "svelte/transition";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import type { TimeSheetReviewersResponse } from "$lib/pocketbase-types";

  let { collectionName }: { collectionName: string } = $props();

  let show = $state(false);
  let itemId = $state("");
  let newViewer = $state("");
  let reviewers = $state([] as TimeSheetReviewersResponse[]);

  function closeModal() {
    show = false;
    itemId = "";
    newViewer = "";
    reviewers = [];
  }

  export async function openModal(id: string) {
    show = true;
    itemId = id;
    reviewers = await pb.collection(collectionName).getFullList<TimeSheetReviewersResponse>({
      filter: pb.filter(`time_sheet="${itemId}"`),
      expand: "reviewer.profiles(uid)",
      sort: "-reviewed",
    });
  }

  async function addViewer() {
    const result = await pb.collection(collectionName).create(
      {
        time_sheet: itemId,
        reviewer: newViewer,
      },
      { returnRecord: true },
    );
    try {
      reviewers = await pb.collection(collectionName).getFullList<TimeSheetReviewersResponse>({
        filter: pb.filter(`time_sheet="${itemId}"`),
        expand: "reviewer.profiles(uid)",
        sort: "-reviewed",
      });
    } catch (error) {
      console.log(error);
    }
    newViewer = "";
  }

  async function deleteViewer(reviewerRecordId: string) {
    // first delete the record from the time_sheet_reviewers collection,
    // then update the reviewers state
    await pb.collection(collectionName).delete(reviewerRecordId);
    reviewers = await pb.collection(collectionName).getFullList<TimeSheetReviewersResponse>({
      filter: pb.filter(`time_sheet="${itemId}"`),
      expand: "reviewer.profiles(uid)",
      sort: "-reviewed",
    });
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
        <h1>Share</h1>
        <h5>{itemId}</h5>
      </div>
      <div class="my-2 flex flex-col items-stretch gap-2 overflow-auto">
        <div class="rounded bg-neutral-700 p-2">
          <h3>{reviewers.length === 0 ? "No " : ""}Viewers</h3>
          {#each reviewers as reviewer}
            <span class="flex items-center gap-1">
              <span>
                {reviewer.expand?.reviewer.expand["profiles(uid)"].surname},
                {reviewer.expand?.reviewer.expand["profiles(uid)"].given_name}
              </span>
              <DsActionButton
                action={() => deleteViewer(reviewer.id)}
                icon="feather:x-circle"
                color="red"
                title="Remove Viewer"
              />
            </span>
          {/each}
        </div>

        <span class="flex items-center gap-1">
          <select name="manager" bind:value={newViewer} class="rounded bg-neutral-700 p-1">
            <option disabled selected>- select manager -</option>
            {#each $globalStore.managers as m (m.id)}
              <option value={m.id}>
                {m.surname}, {m.given_name}
              </option>
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
