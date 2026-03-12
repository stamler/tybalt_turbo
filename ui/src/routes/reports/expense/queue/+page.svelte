<script lang="ts">
  import { resolve } from "$app/paths";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import type { ExpenseCommitQueueRow } from "$lib/svelte-types";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import { shortDate } from "$lib/utilities";
  import type { PageData } from "./$types";
  import { untrack } from "svelte";

  let { data }: { data: PageData } = $props();

  // No counts; page shows all submitted, uncommitted expenses in the commit queue.

  let rows: ExpenseCommitQueueRow[] = $state(untrack(() => data.items || []));
  let rejectModal: RejectModal;
  const hasCommitAccess = $derived(
    $globalStore.showAllUi || $globalStore.claims.includes("commit"),
  );

  async function refreshRows() {
    rows = (await pb.send("/api/expenses/commit_queue", {
      method: "GET",
    })) as ExpenseCommitQueueRow[];
  }

  async function commit(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/commit`, { method: "POST" });
      await refreshRows();
    } catch (error: unknown) {
      const responseError = error as { response?: { error?: string } };
      globalStore.addError(responseError?.response?.error || "Commit failed");
    }
  }

  function openReject(id: string) {
    rejectModal?.openModal(id);
  }

  function canViewDetails(row: ExpenseCommitQueueRow) {
    if ($globalStore.showAllUi) return true;
    if (hasCommitAccess && row.phase === "Approved") return true;
    return false;
  }

  function canReject(row: ExpenseCommitQueueRow) {
    return hasCommitAccess && row.phase === "Approved" && row.rejected === "";
  }

  function detailsHref(id: string) {
    return resolve(`/expenses/${id}/details`);
  }
</script>

<!-- No pay period selector; show all pending items -->
<RejectModal collectionName="expenses" bind:this={rejectModal} on:refresh={() => refreshRows()} />

<DsList items={rows} groupField="phase" inListHeader="Expense Commit Queue">
  {#snippet groupHeader(label)}
    <span class="text-xs tracking-wide text-neutral-600 uppercase">{label}</span>
  {/snippet}
  {#snippet headline(r: ExpenseCommitQueueRow)}
    {#if canViewDetails(r)}
      <a href={resolve(`/expenses/${r.id}/details`)} class="underline"
        >{r.surname}, {r.given_name}</a
      >
    {:else}
      <span>{r.surname}, {r.given_name}</span>
    {/if}
    {#if r.rejected !== ""}
      <DsLabel color="red">Rejected</DsLabel>
    {/if}
  {/snippet}
  {#snippet line1(r: ExpenseCommitQueueRow)}
    <span class="text-sm text-gray-500">
      {#if r.phase === "Approved"}
        Approved by {r.approver_name} on {shortDate(r.approved.split("T")[0], true)}
      {:else if r.phase === "Submitted"}
        Pending approval
      {/if}
    </span>
  {/snippet}
  {#snippet line2(r: ExpenseCommitQueueRow)}
    <div class="text-xs text-neutral-600">
      {#if r.allowance_str}
        {r.allowance_str} ·
      {/if}
      {r.job_number}
      {r.job_description}
      {#if r.client_name}
        &nbsp;for {r.client_name}
      {/if}
    </div>
  {/snippet}
  {#snippet line3(r: ExpenseCommitQueueRow)}
    <div class="text-xs text-neutral-600">
      Date: {shortDate(r.date, true)} · Total: ${r.total?.toFixed(2)}
    </div>
  {/snippet}
  {#snippet actions(r: ExpenseCommitQueueRow)}
    {#if hasCommitAccess && r.phase === "Approved" && r.rejected === ""}
      <DsActionButton action={() => commit(r.id)}>Commit</DsActionButton>
    {/if}
    {#if canReject(r)}
      <DsActionButton action={() => openReject(r.id)}>Reject</DsActionButton>
    {/if}
    {#if canViewDetails(r)}
      <DsActionButton action={() => (window.location.href = detailsHref(r.id))}>View</DsActionButton
      >
    {/if}
  {/snippet}
</DsList>
