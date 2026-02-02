<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import { shortDate } from "$lib/utilities";
  import { untrack } from "svelte";

  let { data }: { data: any } = $props();

  // No counts; page shows all pending items

  type Row = {
    id: string;
    given_name: string;
    surname: string;
    submitted: boolean;
    approved: string;
    rejected: string;
    commited?: string;
    approver_name: string;
    committer_name: string;
    rejector_name: string;
    phase: "Approved" | "Submitted" | "Committed" | "Unsubmitted";
    date: string;
    allowance_str: string;
    job_number: string;
    job_description: string;
    client_name: string;
    total: number;
  };

  let rows: Row[] = $state(untrack(() => data.items || []));

  async function commit(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/commit`, { method: "POST" });
      // Refresh list after commit
      rows = await pb.send("/api/expenses/tracking", { method: "GET" });
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Commit failed");
    }
  }
</script>

<!-- No pay period selector; show all pending items -->

<DsList items={rows} groupField="phase" inListHeader="Expense Commit Queue">
  {#snippet groupHeader(label)}
    <span class="text-xs tracking-wide text-neutral-600 uppercase">{label}</span>
  {/snippet}
  {#snippet headline(r: Row)}
    <a href={`/expenses/${r.id}/details`} class="underline">{r.surname}, {r.given_name}</a>
    {#if r.rejected !== ""}
      <DsLabel color="red">Rejected</DsLabel>
    {/if}
  {/snippet}
  {#snippet line1(r: Row)}
    <span class="text-sm text-gray-500">
      {#if r.phase === "Approved"}
        Approved by {r.approver_name} on {shortDate(r.approved.split("T")[0], true)}
      {:else if r.phase === "Submitted"}
        Pending approval
      {/if}
    </span>
  {/snippet}
  {#snippet line2(r: Row)}
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
  {#snippet line3(r: Row)}
    <div class="text-xs text-neutral-600">
      Date: {shortDate(r.date, true)} · Total: ${r.total?.toFixed(2)}
    </div>
  {/snippet}
  {#snippet actions(r: Row)}
    {#if r.phase === "Approved" && r.rejected === ""}
      <DsActionButton action={() => commit(r.id)}>Commit</DsActionButton>
    {/if}
    <DsActionButton action={() => (window.location.href = `/expenses/${r.id}/details`)}
      >View</DsActionButton
    >
  {/snippet}
</DsList>
