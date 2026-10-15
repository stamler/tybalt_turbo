<script lang="ts">
  import DSCallout from "./DSCallout.svelte";
  import DSPopover from "./DSPopover.svelte";

  // Confirms the "Close project" action (POST /api/jobs/{id}/close) and says
  // what closing will do to the project's referenced proposal, if it has one.

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

  // The server applies the same rules: an Awarded proposal is fine, an
  // imported In Progress or Submitted one is awarded, and any other blocks.
  const autoAwards = $derived(
    proposal !== null &&
      proposal.imported &&
      (proposal.status === "In Progress" || proposal.status === "Submitted"),
  );
  const blocked = $derived(proposal !== null && proposal.status !== "Awarded" && !autoAwards);
</script>

<DSPopover
  bind:show
  title="Close {jobNumber}"
  error={contextError}
  submitting={submitting || loadingContext}
  submitDisabled={blocked}
  submitLabel={loadingContext ? "Loading..." : "Close project"}
  {onSubmit}
  {onCancel}
>
  <p>
    The project's status will be set to Closed, and a note will be added to the client's notes. Its
    billing details do not need to be complete.
  </p>

  {#if loadingContext}
    <p>Checking the referenced proposal...</p>
  {:else if proposal && blocked}
    <DSCallout tone="danger" title="This project can't be closed yet">
      <p>
        Its proposal {proposal.number} is {proposal.status}. Award the proposal before closing the
        project.
      </p>
    </DSCallout>
  {:else if proposal && autoAwards}
    <DSCallout tone="warning" title="Proposal {proposal.number} will be awarded">
      <p>
        It is {proposal.status}. Closing this project also sets the proposal to Awarded and adds a
        note to it.
      </p>
    </DSCallout>
  {/if}
</DSPopover>
