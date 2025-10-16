<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import { shortDate } from "$lib/utilities";
  import { page } from "$app/stores";

  const payPeriodEnding = $derived.by(() => $page.params.payPeriodEnding);
  let rows = $state([] as any[]);

  async function init() {
    try {
      rows = await pb.send(`/api/expenses/tracking/${payPeriodEnding}`, { method: "GET" });
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to load expenses");
    }
  }

  $effect(() => {
    if (payPeriodEnding) {
      init();
    }
  });

  async function commit(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/commit`, { method: "POST" });
      await init();
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Commit failed");
    }
  }
</script>

<DsList items={rows} groupField="phase" inListHeader={`Expenses for ${payPeriodEnding}`}>
  {#snippet groupHeader(label)}
    <span class="text-xs uppercase tracking-wide text-neutral-600">{label}</span>
  {/snippet}
  {#snippet headline(r)}
    <a href={`/expenses/${r.id}/details`} class="underline">{r.surname}, {r.given_name}</a>
    {#if r.rejected !== ""}
      <DsLabel color="red">Rejected</DsLabel>
    {/if}
  {/snippet}
  {#snippet byline(r)}
    <span class="text-sm text-gray-500">
      ${r.total?.toFixed(2)}
    </span>
  {/snippet}
  {#snippet line1(r)}
    <span class="text-sm text-gray-500">
      {#if r.approved !== ""}
        Approved by {r.approver_name || r.approver} on {shortDate(r.approved.split("T")[0], true)}
      {:else if r.submitted && r.approved === ""}
        Pending approval by {r.approver_name || r.approver}
      {/if}
    </span>
  {/snippet}
  {#snippet line2(r)}
    <div class="text-xs text-neutral-600">
      {#if r.committed !== ""}
        <div class="text-sm text-gray-500">
          Committed by {r.committer_name || r.committer} on {shortDate(
            r.committed.split("T")[0],
            true,
          )}
        </div>
      {/if}
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
  {#snippet line3(r)}
    <div class="text-xs text-neutral-600">
      Date: {shortDate(r.date, true)} · Total: ${r.total?.toFixed(2)}
    </div>
  {/snippet}
  {#snippet actions(r)}
    {#if r.phase === "Approved" && r.rejected === ""}
      <DsActionButton action={() => commit(r.id)}>Commit</DsActionButton>
    {/if}
  {/snippet}
</DsList>
