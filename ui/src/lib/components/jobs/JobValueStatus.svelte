<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DSActionButton from "$lib/components/DSActionButton.svelte";
  import DSCallout from "$lib/components/DSCallout.svelte";
  import DSPopover from "$lib/components/DSPopover.svelte";
  import JobValueReview from "./JobValueReview.svelte";
  import { JobValue, valueCAD } from "$lib/jobValue.svelte";
  // A pending or rejected value change needs someone to act, so it shows at the
  // top of the job rather than inside Additional Details.
  let { value }: { value: JobValue } = $props();
  const item = $derived(value.inProgress);
  const claims = $derived($globalStore.claims);
  const isApprover = $derived(
    item?.status === "Pending" &&
      item.assigned_approver === pb.authStore.record?.id &&
      claims.includes("project_authorization"),
  );
  let reviewing = $state(false),
    confirmingDelete = $state(false);
  async function deleteChange() {
    if (item && (await value.act(item, "delete"))) confirmingDelete = false;
  }
</script>

{#if item}
  <DSCallout
    tone={item.status === "Rejected" ? "danger" : "warning"}
    title={`Value change to ${valueCAD(item.project_value)} ${item.status === "Rejected" ? "rejected" : "pending"}`}
  >
    <p>
      {#if item.status === "Rejected"}
        Rejected by {item.rejector_name || item.rejector}: {item.rejection_reason}
      {:else}
        Awaiting {item.assigned_approver_name || item.assigned_approver}
      {/if}
    </p>
    {#snippet actions()}
      {#if isApprover}<DSActionButton variant="primary" action={() => (reviewing = true)}
          >Review</DSActionButton
        >{/if}
      {#if claims.includes("job")}<DSActionButton
          variant="secondary"
          action={() => value.openProposal(item)}>Correct value change</DSActionButton
        >{/if}
      {#if claims.includes("admin")}<DSActionButton
          variant="danger"
          action={() => (confirmingDelete = true)}>Delete pending change</DSActionButton
        >{/if}
    {/snippet}
  </DSCallout>
  {#if value.error && !reviewing && !confirmingDelete}<p role="alert" class="text-sm text-red-800">
      {value.error}
    </p>{/if}

  {#if reviewing}
    <div class="fixed inset-0 z-9999 flex items-center justify-center bg-black/50">
      <div class="w-full max-w-md space-y-4 rounded-lg bg-white p-6 shadow-xl">
        <h2 class="text-xl font-bold">Review value change</h2>
        <JobValueReview {value} {item} ondone={() => (reviewing = false)} />
        {#if value.error}<p role="alert" class="text-sm text-red-700">{value.error}</p>{/if}
        <div class="flex justify-end">
          <button
            type="button"
            class="rounded-sm bg-neutral-200 px-4 py-2 text-neutral-700 hover:bg-neutral-300"
            onclick={() => (reviewing = false)}>Close</button
          >
        </div>
      </div>
    </div>
  {/if}

  <DSPopover
    bind:show={confirmingDelete}
    title="Delete pending value change?"
    subtitle={`This deletes the proposed change to ${valueCAD(item.project_value)} and its PDF.`}
    error={value.error || null}
    submitting={value.busy}
    submitLabel="Delete pending change"
    danger
    onSubmit={deleteChange}
  >
    <p class="rounded-sm border border-red-300 bg-red-50 p-3 text-sm text-red-900">
      The project value stays {valueCAD(item.effective_project_value)}. This cannot be undone.
    </p>
  </DSPopover>
{/if}
