<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import { jobs } from "$lib/stores/jobs";
  import type { JobApiResponse } from "$lib/stores/jobs";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import CloseProjectConfirmPopover from "$lib/components/CloseProjectConfirmPopover.svelte";
  import { jobRequestsEnabled, jobsEditingEnabled } from "$lib/stores/appConfig";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import Icon from "@iconify/svelte";
  import { goto } from "$app/navigation";

  // initialize the jobs store, noop if already initialized
  jobs.init();

  // Track which job is currently being validated for project creation
  let validatingJobId = $state<string | null>(null);
  let closingJobId = $state<string | null>(null);
  let showCloseConfirm = $state(false);
  let pendingCloseJob = $state<JobApiResponse | null>(null);
  let closeContextLoading = $state(false);
  let closeContextError = $state<string | null>(null);
  let closeProposal = $state<{
    id: string;
    number: string;
    status: string;
    imported: boolean;
  } | null>(null);

  // Validate proposal and redirect to create project or edit page
  async function handleCreateReferencingProject(jobId: string) {
    if ($jobRequestsEnabled) {
      await goto(`/job-requests/new?proposal=${jobId}`);
      return;
    }
    if (validatingJobId !== null) return;
    validatingJobId = jobId;
    try {
      const response = await pb.send(`/api/jobs/${jobId}/validate-proposal`, {
        method: "GET",
      });
      if (response.valid) {
        // Proposal is valid - redirect to create project with today's award date
        await goto(`/jobs/add/from/${jobId}?setAwardToday=true`);
      } else {
        // Proposal has validation errors - store errors and redirect flag, then go to edit page
        if (typeof sessionStorage !== "undefined") {
          if (response.errors) {
            sessionStorage.setItem(
              `proposal_validation_errors_${jobId}`,
              JSON.stringify(response.errors),
            );
          }
          // Flag to redirect back to create project page after successful save
          sessionStorage.setItem(`redirect_to_create_project_${jobId}`, "true");
        }
        await goto(`/jobs/${jobId}/edit`);
      }
    } catch (e) {
      console.error("Failed to validate proposal", e);
      // On error, redirect to edit page as a fallback
      await goto(`/jobs/${jobId}/edit`);
    } finally {
      validatingJobId = null;
    }
  }

  // Toggle: "projects" or "proposals" - default to projects
  let jobType = $state<"projects" | "proposals">("projects");

  const jobTypeOptions = [
    { id: "projects", label: "Projects" },
    { id: "proposals", label: "Proposals" },
  ];

  // Filter function for search results based on job type
  const jobTypeFilter = $derived((job: JobApiResponse) => {
    const isProposal = job.number?.startsWith("P");
    return jobType === "proposals" ? isProposal : !isProposal;
  });

  async function downloadJson() {
    try {
      // Use a date from the past to get all non-imported jobs
      const updatedAfter = "2000-01-01";
      const data = await pb.send(`/api/export_legacy/jobs/${updatedAfter}`, { method: "GET" });
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `jobs_export_${new Date().toISOString().split("T")[0]}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to download JSON");
    }
  }

  // Open confirmation first, then resolve proposal context so the modal can
  // show exactly what side effects may occur before the user confirms.
  async function openCloseConfirm(job: JobApiResponse) {
    pendingCloseJob = job;
    showCloseConfirm = true;
    closeContextLoading = true;
    closeContextError = null;
    closeProposal = null;

    try {
      const jobDetails = await pb.send(`/api/jobs/${job.id}/details`, { method: "GET" });
      const proposalId = jobDetails?.proposal_id as string | undefined;
      if (proposalId) {
        const proposal = await pb.send(`/api/jobs/${proposalId}`, { method: "GET" });
        closeProposal = {
          id: proposal.id,
          number: (jobDetails?.proposal_number as string) || proposal.number || proposalId,
          status: proposal.status,
          imported: Boolean(proposal.imported),
        };
      }
    } catch (error: any) {
      closeContextError =
        error?.response?.message ??
        error?.response?.error ??
        error?.message ??
        "Unable to load referenced proposal details.";
    } finally {
      closeContextLoading = false;
    }
  }

  function cancelCloseConfirm() {
    showCloseConfirm = false;
    pendingCloseJob = null;
    closeProposal = null;
    closeContextError = null;
    closeContextLoading = false;
  }

  // Close an Active project through /api/jobs/{id}/close rather than patching
  // the record, so the backend applies the close, its notes and any proposal
  // auto-award in one transaction, the same as the job page.
  async function handleCloseConfirmSubmit() {
    if (!pendingCloseJob || closingJobId !== null || closeContextLoading) return;
    closingJobId = pendingCloseJob.id;
    try {
      await pb.send(`/api/jobs/${pendingCloseJob.id}/close`, { method: "POST" });
      await jobs.refresh(pendingCloseJob.id);
      cancelCloseConfirm();
    } catch (error: any) {
      const msg =
        error?.response?.message ??
        error?.response?.error ??
        error?.message ??
        "Failed to close project";
      globalStore.addError(msg);
    } finally {
      closingJobId = null;
    }
  }
</script>

{#if $jobs.index !== null}
  <DsSearchList
    index={$jobs.index}
    filter={jobTypeFilter}
    inListHeader={jobType === "projects" ? "Projects" : "Proposals"}
    fieldName="job"
    uiName="search jobs..."
  >
    {#snippet searchBarExtra()}
      <div class="flex items-center gap-2 max-[639px]:w-full max-[639px]:flex-wrap">
        <DSToggle bind:value={jobType} options={jobTypeOptions} ariaLabel="Job type filter" />
        <button
          onclick={downloadJson}
          class="flex items-center gap-1 rounded-sm bg-neutral-200 px-3 py-1 text-sm text-gray-700 hover:bg-neutral-300"
        >
          <Icon icon="mdi:download" class="text-base" /> JSON
        </button>
      </div>
    {/snippet}
    {#snippet anchor({ id, number }: JobApiResponse)}
      <a href="/jobs/{id}/details" class="font-bold hover:underline">{number}</a>
    {/snippet}
    {#snippet headline({ description }: JobApiResponse)}{description}{/snippet}
    {#snippet byline({ client }: JobApiResponse)}{client}{/snippet}
    {#snippet line1({ branch, manager }: JobApiResponse)}{#if branch}<DsLabel color="neutral"
          >{branch}</DsLabel
        >{/if}{#if manager}<DsLabel color="purple">{manager}</DsLabel>{/if}{/snippet}
    {#snippet actions(job: JobApiResponse)}
      {@const id = job.id}
      {@const number = job.number}
      {#if number?.startsWith("P") && ($jobRequestsEnabled ? $globalStore.claims.includes("time") : $globalStore.claims.includes("job"))}
        <DsActionButton
          action={() => handleCreateReferencingProject(id)}
          icon="mdi:briefcase-plus"
          title={$jobRequestsEnabled
            ? "Create Project Request from Proposal"
            : "Create referencing project"}
          color="yellow"
          loading={validatingJobId === id}
        />
      {/if}
      {#if !number?.startsWith("P") && job.status === "Active" && $jobsEditingEnabled && $globalStore.claims.includes("job")}
        <DsActionButton
          action={() => openCloseConfirm(job)}
          icon="mdi:archive-check"
          title="Close project"
          color="red"
          loading={closingJobId === id}
        />
      {/if}
      {#if $globalStore.claims.includes("job")}<DsActionButton
          action="/jobs/{id}/edit"
          icon="mdi:pencil"
          title="Edit"
          color="blue"
        />{/if}
    {/snippet}
  </DsSearchList>
{/if}

<CloseProjectConfirmPopover
  bind:show={showCloseConfirm}
  jobNumber={pendingCloseJob?.number ?? "Project"}
  proposal={closeProposal}
  loadingContext={closeContextLoading}
  contextError={closeContextError}
  submitting={closingJobId !== null}
  onSubmit={handleCloseConfirmSubmit}
  onCancel={cancelCloseConfirm}
/>
