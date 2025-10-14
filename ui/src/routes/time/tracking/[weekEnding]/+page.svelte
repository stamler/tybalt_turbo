<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import type { SvelteComponent } from "svelte";
  import { page } from "$app/stores";

  const weekEnding = $derived.by(() => $page.params.weekEnding);
  let rows = $state([] as any[]);
  let rejectModal: SvelteComponent;

  async function init() {
    try {
      rows = await pb.send(`/api/time_sheets/tracking/weeks/${weekEnding}`, { method: "GET" });
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to load timesheets");
    }
  }

  $effect(() => {
    if (weekEnding) {
      init();
    }
  });

  async function commit(id: string) {
    try {
      await pb.send(`/api/time_sheets/${id}/commit`, { method: "POST" });
      await init();
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Commit failed");
    }
  }

  function openReject(id: string) {
    // @ts-ignore exported function on the component instance
    rejectModal.openModal(id);
  }
</script>

<RejectModal collectionName="time_sheets" bind:this={rejectModal} on:refresh={() => init()} />

<DsList items={rows} inListHeader={`Time Tracking for ${weekEnding}`}>
  {#snippet headline(r)}
    <a href={`/time/sheets/${r.id}/details`} class="underline">{r.surname}, {r.given_name}</a>
  {/snippet}
  {#snippet line1(r)}
    <span class="text-sm text-gray-500"
      >Approved: {r.approved || "-"} | Rejected: {r.rejected || "-"} | Committed: {r.committed ||
        "-"}</span
    >
  {/snippet}
  {#snippet actions(r)}
    <DsActionButton action={() => commit(r.id)}>Commit</DsActionButton>
    <DsActionButton action={() => openReject(r.id)}>Reject</DsActionButton>
  {/snippet}
</DsList>
