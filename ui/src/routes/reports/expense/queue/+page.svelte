<script lang="ts">
  import { resolve } from "$app/paths";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
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

  function canReject(row: ExpenseCommitQueueRow) {
    return hasCommitAccess && row.rejected === "";
  }

  function detailsHref(id: string) {
    return resolve(`/expenses/${id}/details`);
  }

  function attachmentHref(id: string, attachment: string) {
    return `${PUBLIC_POCKETBASE_URL}/api/files/expenses/${id}/${attachment}`;
  }

  function openAttachment(id: string, attachment: string) {
    window.open(attachmentHref(id, attachment), "_blank", "noopener,noreferrer");
  }
</script>

<!-- No pay period selector; show all pending items -->
<RejectModal collectionName="expenses" bind:this={rejectModal} on:refresh={() => refreshRows()} />

<DsList items={rows} inListHeader="Expense Commit Queue">
  {#snippet headline(r: ExpenseCommitQueueRow)}
    <a href={resolve(`/expenses/${r.id}/details`)} class="underline">{r.surname}, {r.given_name}</a>
    {#if r.rejected !== ""}
      <DsLabel color="red">Rejected</DsLabel>
    {/if}
  {/snippet}
  {#snippet line1(r: ExpenseCommitQueueRow)}
    <span class="text-sm text-gray-500">
      Approved by {r.approver_name} on {shortDate(r.approved.split("T")[0], true)}
      {#if r.rejected !== ""}
        · Rejected on {shortDate(r.rejected.split("T")[0], true)}
      {/if}
    </span>
  {/snippet}
  {#snippet line2(r: ExpenseCommitQueueRow)}
    <div class="text-xs text-neutral-600">
      {#if r.allowance_str}
        {r.allowance_str}
      {:else}
        {r.description}
      {/if}
      {#if r.job_number !== ""}
        &nbsp;&middot;&nbsp;
        {r.job_number}
        {r.job_description}
        {#if r.client_name}
          &nbsp;/ {r.client_name}
        {/if}
      {/if}
    </div>
  {/snippet}
  {#snippet line3(r: ExpenseCommitQueueRow)}
    <div class="flex items-center gap-2 text-xs text-neutral-600">
      <span>Date: {shortDate(r.date, true)} · Total: ${r.total?.toFixed(2)}</span>
      {#if r.attachment !== ""}
        <DsActionButton
          action={() => openAttachment(r.id, r.attachment)}
          title="Open attachment in a new tab"
        >
          Attachment
        </DsActionButton>
      {/if}
    </div>
  {/snippet}
  {#snippet actions(r: ExpenseCommitQueueRow)}
    {#if hasCommitAccess && r.rejected === ""}
      <DsActionButton action={() => commit(r.id)}>Commit</DsActionButton>
    {/if}
    {#if canReject(r)}
      <DsActionButton action={() => openReject(r.id)}>Reject</DsActionButton>
    {/if}
    <DsActionButton action={() => (window.location.href = detailsHref(r.id))}>View</DsActionButton>
  {/snippet}
</DsList>
