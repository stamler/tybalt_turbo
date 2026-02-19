<script lang="ts">
  import DSPopover from "./DSPopover.svelte";

  type ProposalContext = {
    id: string;
    number: string;
    status: string;
    imported: boolean;
  };

  let {
    show = $bindable(),
    jobNumber,
    proposal = null,
    loadingContext = false,
    contextError = null,
    submitting = false,
    onSubmit,
    onCancel,
  }: {
    show: boolean;
    jobNumber: string;
    proposal?: ProposalContext | null;
    loadingContext?: boolean;
    contextError?: string | null;
    submitting?: boolean;
    onSubmit: () => void;
    onCancel: () => void;
  } = $props();

  const isTerminalProposalStatus = (status: string) =>
    status === "Not Awarded" || status === "Cancelled" || status === "No Bid";
  const isAutoAwardEligibleStatus = (status: string) =>
    status === "In Progress" || status === "Submitted";
</script>

<DSPopover
  bind:show
  title="Confirm Fast Close: {jobNumber}"
  subtitle="This action is intended for imported legacy projects and will immediately attempt to close the project."
  error={contextError}
  submitting={submitting || loadingContext}
  submitLabel={loadingContext ? "Loading..." : "Close Project"}
  {onSubmit}
  {onCancel}
>
  <div class="rounded-sm border border-amber-300 bg-amber-50 p-3 text-amber-900">
    <p class="font-semibold">Review what will happen if you continue:</p>
    <ul class="mt-2 list-disc space-y-1 pl-5 text-sm">
      <li>Project status will be set to <strong>Closed</strong>.</li>
      <li>Project <code>_imported</code> will be set to <strong>false</strong>.</li>
      <li>A project client note will be created automatically.</li>
    </ul>
  </div>

  {#if loadingContext}
    <p class="text-sm text-neutral-600">Loading referenced proposal details...</p>
  {:else if proposal}
    <div class="rounded-sm border border-neutral-300 bg-neutral-50 p-3 text-sm text-neutral-800">
      <p class="font-semibold">Referenced proposal detected: {proposal.number}</p>
      <p class="mt-1">
        Current proposal state:
        <strong>status={proposal.status}</strong>,
        <strong>imported={proposal.imported ? "true" : "false"}</strong>.
      </p>
      <ul class="mt-2 list-disc space-y-1 pl-5">
        <li>If proposal is already <strong>Awarded</strong>, close proceeds.</li>
        <li>
          If proposal is imported and status is <strong>In Progress</strong> or
          <strong>Submitted</strong>, it will be auto-awarded, set to
          <code>_imported=false</code>, and a proposal note will be created.
        </li>
        <li>
          If proposal is <strong>Not Awarded</strong>, <strong>Cancelled</strong>, or
          <strong>No Bid</strong>, close is blocked and no changes are saved.
        </li>
      </ul>
      {#if proposal.imported && isAutoAwardEligibleStatus(proposal.status)}
        <p class="mt-2 font-semibold text-amber-800">
          This close attempt will auto-award the referenced proposal.
        </p>
      {:else if isTerminalProposalStatus(proposal.status)}
        <p class="mt-2 font-semibold text-red-700">
          This close attempt will be rejected due to terminal proposal status.
        </p>
      {/if}
    </div>
  {:else}
    <p class="text-sm text-neutral-700">
      No referenced proposal is linked to this project. Only project close changes will be applied.
    </p>
  {/if}
</DSPopover>
