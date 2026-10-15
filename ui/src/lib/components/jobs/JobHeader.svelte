<script lang="ts">
  import DSActionButton from "$lib/components/DSActionButton.svelte";
  import DSMenuItem from "$lib/components/DSMenuItem.svelte";
  import DSPageHeader from "$lib/components/DSPageHeader.svelte";
  import DSPopover from "$lib/components/DSPopover.svelte";
  import DSStatusLabel from "$lib/components/DSStatusLabel.svelte";
  import DSTextInput from "$lib/components/DSTextInput.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DSCallout from "$lib/components/DSCallout.svelte";
  import FastCloseConfirmPopover from "$lib/components/FastCloseConfirmPopover.svelte";
  import ChangeClientPopover from "$lib/components/jobs/ChangeClientPopover.svelte";
  import { pb } from "$lib/pocketbase";
  import { goto, invalidateAll } from "$app/navigation";
  import { resolve } from "$app/paths";
  import { ClientResponseError } from "pocketbase";
  import { globalStore } from "$lib/stores/global";
  import { jobsEditingEnabled, jobRequestsEnabled } from "$lib/stores/appConfig";
  import { JobsStatusOptions } from "$lib/pocketbase-types";
  import { saveErrorMessage } from "$lib/utilities";
  import type { JobDetails } from "./types";

  // The top of the job page: number, description, status, links to the parent
  // and proposal, and every action on the job. Frequent actions are buttons;
  // rare and admin ones are in the "⋯" menu. Key this component on the job id
  // so loading and dialog state reset when moving between jobs.
  let { job }: { job: JobDetails } = $props();

  const claims = $derived($globalStore.claims);
  const isProposal = $derived(job.number?.startsWith("P") ?? false);
  const currentUserID = $derived(pb.authStore.record?.id ?? "");
  // With request approval, holders of time start projects through a request.
  const canStartProject = $derived(
    $jobRequestsEnabled ? claims.includes("time") : claims.includes("job"),
  );
  const canEdit = $derived(
    claims.includes("job") ||
      (currentUserID !== "" &&
        (job.manager?.id === currentUserID || job.alternate_manager?.id === currentUserID)),
  );
  // Only a top-level project number can have subjobs.
  const canCreateSubjob = $derived(
    !isProposal && /^\d{2}-\d{3,4}$/.test(job.number ?? "") && canStartProject,
  );
  const canSetNumber = $derived($jobsEditingEnabled && claims.includes("admin"));
  // A subjob uses its parent's client, so only a top-level project's client
  // can be changed, and it moves its subjobs with it.
  const canChangeClient = $derived(
    !isProposal &&
      !job.parent_number &&
      Boolean(job.client?.id) &&
      $jobsEditingEnabled &&
      claims.includes("job"),
  );
  const canFastCloseImportedProject = $derived(
    !isProposal &&
      job.status === "Active" &&
      job.imported === true &&
      $jobsEditingEnabled &&
      claims.includes("job"),
  );
  const showProposalLink = $derived(!isProposal && Boolean(job.proposal_id));
  const hasNumberHierarchyWarning = $derived(
    Boolean(job.parent_number) || (Array.isArray(job.children) && job.children.length > 0),
  );

  // Proposal actions --------------------------------------------------------

  let validatingForProject = $state(false);

  // Validate the proposal, then go to create the project, or to the edit page
  // to fix what is missing first.
  async function createReferencingProject() {
    if ($jobRequestsEnabled) {
      // eslint-disable-next-line svelte/no-navigation-without-resolve -- query on a resolved path
      await goto(`${resolve("/job-requests/new")}?proposal=${job.id}`);
      return;
    }
    if (validatingForProject) return;
    validatingForProject = true;
    try {
      const response = await pb.send(`/api/jobs/${job.id}/validate-proposal`, {
        method: "GET",
      });
      if (response.valid) {
        // Proposal is valid - redirect to create project with today's award date
        await goto(
          // eslint-disable-next-line svelte/no-navigation-without-resolve -- query on a resolved path
          `${resolve("/jobs/add/from/[proposal]", { proposal: job.id })}?setAwardToday=true`,
        );
      } else {
        // Proposal has validation errors - store errors and redirect flag, then go to edit page
        if (typeof sessionStorage !== "undefined") {
          if (response.errors) {
            sessionStorage.setItem(
              `proposal_validation_errors_${job.id}`,
              JSON.stringify(response.errors),
            );
          }
          // Flag to redirect back to create project page after successful save
          sessionStorage.setItem(`redirect_to_create_project_${job.id}`, "true");
        }
        await goto(resolve("/jobs/[jid]/edit", { jid: job.id }));
      }
    } catch (e) {
      console.error("Failed to validate proposal", e);
      // On error, redirect to edit page as a fallback
      await goto(resolve("/jobs/[jid]/edit", { jid: job.id }));
    } finally {
      validatingForProject = false;
    }
  }

  let awardError = $state<string | null>(null);

  async function awardProposal() {
    awardError = null;
    try {
      await pb.collection("jobs").update(job.id, { status: JobsStatusOptions.Awarded });
      // Re-run the page load so the "Create Project from Proposal" action
      // appears without a full reload.
      await invalidateAll();
    } catch (e) {
      console.error("Failed to award proposal", e);
      awardError = "Failed to award proposal. Please try again.";
    }
  }

  // Change number -----------------------------------------------------------

  let showSetNumber = $state(false);
  let setNumberSubmitting = $state(false);
  let setNumberValue = $state("");
  let setNumberError = $state<string | null>(null);
  let setNumberErrors = $state({} as Record<string, { message: string }>);

  function openSetNumber() {
    setNumberValue = job.number ?? "";
    setNumberError = null;
    setNumberErrors = {};
    showSetNumber = true;
  }

  function closeSetNumber() {
    if (setNumberSubmitting) return;
    showSetNumber = false;
    setNumberError = null;
    setNumberErrors = {};
  }

  async function submitSetNumber() {
    if (setNumberSubmitting) return;
    setNumberSubmitting = true;
    setNumberError = null;
    setNumberErrors = {};
    try {
      await pb.send(`/api/jobs/${job.id}/set-number`, {
        method: "POST",
        body: { number: setNumberValue },
      });
      setNumberSubmitting = false;
      closeSetNumber();
      await invalidateAll();
    } catch (error: unknown) {
      const response = error instanceof ClientResponseError ? error.data : {};
      const backendErrors = response?.data as Record<string, { message: string }> | undefined;
      setNumberErrors = backendErrors ?? {};
      if (!backendErrors || Object.keys(backendErrors).length === 0) {
        setNumberError = response?.message ?? "Failed to change job number";
      }
    } finally {
      setNumberSubmitting = false;
    }
  }

  // Change client -----------------------------------------------------------

  let showChangeClient = $state(false);

  // Fast close of an imported project ---------------------------------------

  let showFastClose = $state(false);
  let fastCloseLoading = $state(false);
  let fastCloseError = $state<string | null>(null);
  let fastCloseSubmitting = $state(false);
  let fastCloseProposal = $state<{
    id: string;
    number: string;
    status: string;
    imported: boolean;
  } | null>(null);

  // Load the referenced proposal first so the confirmation can show exactly
  // what closing will do to it.
  async function openFastClose() {
    showFastClose = true;
    fastCloseLoading = true;
    fastCloseError = null;
    fastCloseProposal = null;
    try {
      if (job.proposal_id) {
        const proposal = await pb.send(`/api/jobs/${job.proposal_id}`, { method: "GET" });
        fastCloseProposal = {
          id: proposal.id,
          number: job.proposal_number || proposal.number || job.proposal_id,
          status: proposal.status,
          imported: Boolean(proposal.imported),
        };
      }
    } catch (error: unknown) {
      fastCloseError = saveErrorMessage(error, "Unable to load referenced proposal details.");
    } finally {
      fastCloseLoading = false;
    }
  }

  function closeFastClose() {
    showFastClose = false;
    fastCloseLoading = false;
    fastCloseError = null;
    fastCloseProposal = null;
  }

  // Closes through /api/jobs/{id}/close rather than patching the record, so
  // the backend applies the close, its notes and any proposal auto-award in
  // one transaction, the same policy as the jobs list.
  async function submitFastClose() {
    if (fastCloseSubmitting || fastCloseLoading) return;
    fastCloseSubmitting = true;
    try {
      await pb.send(`/api/jobs/${job.id}/close`, { method: "POST" });
      await invalidateAll();
      closeFastClose();
    } catch (error: unknown) {
      globalStore.addError(saveErrorMessage(error, "Failed to close imported project"));
    } finally {
      fastCloseSubmitting = false;
    }
  }
</script>

{#snippet jobContext()}
  {#if job.parent_id}
    Subjob of <a class="link" href={resolve("/jobs/[jid]/details", { jid: job.parent_id })}
      >{job.parent_number || job.parent_id}</a
    >
  {/if}
  {#if job.parent_id && showProposalLink}·{/if}
  {#if showProposalLink}
    From proposal <a class="link" href={resolve("/jobs/[jid]/details", { jid: job.proposal_id })}
      >{job.proposal_number || job.proposal_id}</a
    >
  {/if}
{/snippet}

<DSPageHeader
  title={job.number}
  subtitle={job.description}
  context={job.parent_id || showProposalLink ? jobContext : undefined}
>
  {#snippet badges()}
    {#if job.status}<DSStatusLabel status={job.status} kind="job" />{/if}
    {#if isProposal}<DsLabel color="yellow">Proposal</DsLabel>{/if}
    {#if job.time_and_materials}<DsLabel color="purple" title="Time and materials">T&M</DsLabel
      >{/if}
  {/snippet}
  {#snippet actions()}
    {#if isProposal && canStartProject && job.status !== "Awarded"}
      <DSActionButton
        action={createReferencingProject}
        icon="mdi:briefcase-plus"
        title={$jobRequestsEnabled
          ? "Create Project Request from Proposal"
          : "Create referencing project"}
        color="yellow"
        loading={validatingForProject}
      />
    {/if}
    {#if isProposal && job.status === "Awarded" && canStartProject}
      <DSActionButton
        action={$jobRequestsEnabled
          ? `/job-requests/new?proposal=${job.id}`
          : `/jobs/add/from/${job.id}`}
        icon="mdi:plus"
        title={$jobRequestsEnabled
          ? "Create Project Request from Proposal"
          : "Create Project from Proposal"}
        color="green"
      />
    {:else if isProposal && job.status === "Submitted" && claims.includes("job")}
      <DSActionButton
        action={awardProposal}
        icon="mdi:trophy"
        title="Award proposal"
        color="yellow"
      />
    {/if}
    {#if canCreateSubjob}
      <DSActionButton
        action={$jobRequestsEnabled ? `/job-requests/new?parent=${job.id}` : `/jobs/add/${job.id}`}
        icon="mdi:file-tree"
        title={$jobRequestsEnabled ? "Create Subjob Request" : "Create Subjob"}
        color="green"
      />
    {/if}
    {#if canEdit}
      <DSActionButton
        action={`/jobs/${job.id}/edit`}
        icon="mdi:pencil"
        title={claims.includes("job") ? "Edit Job" : "Edit allocations"}
        color="blue"
      />
    {/if}
  {/snippet}
  {#snippet menu()}
    {#if canSetNumber}
      <DSMenuItem label="Change number" icon="mdi:pound" action={openSetNumber} />
    {/if}
    {#if canChangeClient}
      <DSMenuItem
        label="Change client"
        icon="mdi:account-switch"
        action={() => (showChangeClient = true)}
      />
    {/if}
    {#if canFastCloseImportedProject}
      <DSMenuItem
        label="Close imported project"
        icon="mdi:archive-check"
        tone="danger"
        action={openFastClose}
      />
    {/if}
  {/snippet}
</DSPageHeader>

{#if awardError}
  <DSCallout tone="danger">{awardError}</DSCallout>
{/if}

<DSPopover
  bind:show={showSetNumber}
  title="Change Job Number"
  subtitle="Admins can manually override the stored job number without reopening the full editor."
  error={setNumberError}
  submitting={setNumberSubmitting}
  submitLabel="Save Number"
  onSubmit={submitSetNumber}
  onCancel={closeSetNumber}
>
  <DSTextInput
    bind:value={setNumberValue}
    errors={setNumberErrors}
    fieldName="number"
    uiName="Job Number"
    disabled={setNumberSubmitting}
  />
  {#if hasNumberHierarchyWarning}
    <DSCallout tone="warning" title="Related records are not renumbered automatically.">
      {#if job.parent_number}
        <p>This job has parent number {job.parent_number}.</p>
      {/if}
      {#if Array.isArray(job.children) && job.children.length > 0}
        <p>
          This job has {job.children.length} child job{job.children.length === 1 ? "" : "s"}.
        </p>
      {/if}
      <p>If related job numbers should stay aligned, update those records separately.</p>
    </DSCallout>
  {/if}
</DSPopover>

<FastCloseConfirmPopover
  bind:show={showFastClose}
  jobNumber={job.number ?? "Project"}
  proposal={fastCloseProposal}
  loadingContext={fastCloseLoading}
  contextError={fastCloseError}
  submitting={fastCloseSubmitting}
  onSubmit={submitFastClose}
  onCancel={closeFastClose}
/>

{#if canChangeClient && showChangeClient}
  <ChangeClientPopover
    bind:show={showChangeClient}
    {job}
    subjobs={job.children ?? []}
    onchanged={() => invalidateAll()}
  />
{/if}
