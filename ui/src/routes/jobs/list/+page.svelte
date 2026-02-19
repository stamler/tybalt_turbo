<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import { jobs } from "$lib/stores/jobs";
  import type { JobApiResponse } from "$lib/stores/jobs";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import FastCloseConfirmPopover from "$lib/components/FastCloseConfirmPopover.svelte";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import Icon from "@iconify/svelte";
  import { goto } from "$app/navigation";

  // initialize the jobs store, noop if already initialized
  jobs.init();

  // Track which job is currently being validated for project creation
  let validatingJobId = $state<string | null>(null);
  let closingJobId = $state<string | null>(null);
  let showFastCloseConfirm = $state(false);
  let pendingFastCloseJob = $state<JobApiResponse | null>(null);
  let fastCloseContextLoading = $state(false);
  let fastCloseContextError = $state<string | null>(null);
  let fastCloseProposal = $state<{
    id: string;
    number: string;
    status: string;
    imported: boolean;
  } | null>(null);

  // Validate proposal and redirect to create project or edit page
  async function handleCreateReferencingProject(jobId: string) {
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
  async function openFastCloseConfirm(job: JobApiResponse) {
    pendingFastCloseJob = job;
    showFastCloseConfirm = true;
    fastCloseContextLoading = true;
    fastCloseContextError = null;
    fastCloseProposal = null;

    try {
      const jobDetails = await pb.send(`/api/jobs/${job.id}/details`, { method: "GET" });
      const proposalId = jobDetails?.proposal_id as string | undefined;
      if (proposalId) {
        const proposal = await pb.send(`/api/jobs/${proposalId}`, { method: "GET" });
        fastCloseProposal = {
          id: proposal.id,
          number: (jobDetails?.proposal_number as string) || proposal.number || proposalId,
          status: proposal.status,
          imported: Boolean(proposal.imported),
        };
      }
    } catch (error: any) {
      fastCloseContextError =
        error?.response?.message ??
        error?.response?.error ??
        error?.message ??
        "Unable to load referenced proposal details.";
    } finally {
      fastCloseContextLoading = false;
    }
  }

  function closeFastCloseConfirm() {
    showFastCloseConfirm = false;
    pendingFastCloseJob = null;
    fastCloseProposal = null;
    fastCloseContextError = null;
    fastCloseContextLoading = false;
  }

  // Close an imported Active project using the dedicated fast-close endpoint.
  //
  // Why this exists:
  // - We intentionally avoid using the generic edit/save route here because the
  //   fast-close policy is specific: imported legacy projects may bypass full
  //   validation, while non-imported projects remain strict.
  // - Calling the dedicated endpoint keeps that policy explicit and prevents
  //   accidental behavior changes in the regular editor flow.
  async function handleFastCloseConfirmSubmit() {
    if (!pendingFastCloseJob || closingJobId !== null || fastCloseContextLoading) return;
    closingJobId = pendingFastCloseJob.id;
    try {
      await pb.send(`/api/jobs/${pendingFastCloseJob.id}/close`, { method: "POST" });
      await jobs.refresh(pendingFastCloseJob.id);
      closeFastCloseConfirm();
    } catch (error: any) {
      const msg =
        error?.response?.message ??
        error?.response?.error ??
        error?.message ??
        "Failed to close imported project";
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
      {#if number?.startsWith("P")}
        <DsActionButton
          action={() => handleCreateReferencingProject(id)}
          icon="mdi:briefcase-plus"
          title="Create referencing project"
          color="yellow"
          loading={validatingJobId === id}
        />
      {/if}
      {#if !number?.startsWith("P") && job.status === "Active" && job.imported}
        <DsActionButton
          action={() => openFastCloseConfirm(job)}
          icon="mdi:archive-check"
          title="Close imported project"
          color="red"
          loading={closingJobId === id}
        />
      {/if}
      <DsActionButton action="/jobs/{id}/edit" icon="mdi:pencil" title="Edit" color="blue" />
    {/snippet}
  </DsSearchList>
{/if}

<FastCloseConfirmPopover
  bind:show={showFastCloseConfirm}
  jobNumber={pendingFastCloseJob?.number ?? "Project"}
  proposal={fastCloseProposal}
  loadingContext={fastCloseContextLoading}
  contextError={fastCloseContextError}
  submitting={closingJobId !== null}
  onSubmit={handleFastCloseConfirmSubmit}
  onCancel={closeFastCloseConfirm}
/>
