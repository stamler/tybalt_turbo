<script lang="ts">
  import JobValueCorrectionFields from "./JobValueCorrectionFields.svelte";
  import { pb } from "$lib/pocketbase";
  import { goto, replaceState } from "$app/navigation";
  import { onMount } from "svelte";
  import {
    clearClientWorkflowDraft,
    type ClientWorkflowTarget,
    keepClientWorkflowDraft,
    readClientWorkflowDraft,
  } from "$lib/clientWorkflowNavigation";
  import { workflowError, type WorkflowRecord } from "$lib/clientEditors";
  import { JobValue, valueCAD } from "$lib/jobValue.svelte";
  // The form for proposing a new project value or correcting a pending or
  // rejected one. It opens as a modal when value.proposing is set.
  let { value: jobValue, effectiveValue }: { value: JobValue; effectiveValue: number } = $props();
  const jobId = $derived(jobValue.jobId);
  let busy = $state(false),
    error = $state("");
  let value = $state<number | undefined>(),
    approver = $state(""),
    documentFile = $state<File | null>(null),
    approvers = $state<WorkflowRecord[]>([]);
  let correctionJob = $state<WorkflowRecord | null>(null),
    correctionOptions = $state<WorkflowRecord>({}),
    correctionAllocations = $state<WorkflowRecord[]>([]),
    correctionLoading = $state(false),
    originalRate = $state(""),
    jobUpdated = $state(""),
    jobReviewToken = $state(""),
    originalAllocations = $state(""),
    correctionStale = $state(false);
  const editing = $derived(jobValue.editing);
  function allocationSignature(rows: WorkflowRecord[]) {
    return JSON.stringify(
      rows
        .map((row) => ({ division: row.division, hours: Number(row.hours) || 0 }))
        .sort((a, b) => a.division.localeCompare(b.division)),
    );
  }
  const correctionFields = [
    "status",
    "description",
    "contact",
    "invoicing_information",
    "manager",
    "alternate_manager",
    "branch",
    "rate_sheet",
    "location",
    "project_award_date",
    "project_completion_date",
    "job_owner",
    "fn_agreement",
    "time_and_materials",
    "client_po",
    "client_reference_number",
  ];
  type ValueDraft = {
    jobId?: string;
    editing: WorkflowRecord | null;
    value?: number;
    approver: string;
    documentFile: File | null;
    documentName: string;
    correctionJob: WorkflowRecord;
    correctionAllocations: WorkflowRecord[];
    originalRate: string;
    jobUpdated: string;
    jobReviewToken: string;
    originalAllocations: string;
  };
  // Restore a form the user left to add a client contact or invoicing profile.
  let restored = false;
  onMount(() => {
    const draft = readClientWorkflowDraft<ValueDraft>(
      `job-value:${jobId || "queue"}`,
      pb.authStore.record?.id || "",
    );
    if (!draft || draft.jobId !== jobId) return;
    restored = true;
    jobValue.openProposal(draft.editing);
    value = draft.value;
    approver = draft.approver;
    documentFile = draft.documentFile instanceof File ? draft.documentFile : null;
    correctionJob = draft.correctionJob;
    correctionAllocations = draft.correctionAllocations;
    originalRate = draft.originalRate;
    jobUpdated = draft.jobUpdated;
    jobReviewToken = draft.jobReviewToken;
    originalAllocations = draft.originalAllocations;
    if (draft.documentName && !documentFile)
      error = "Select the PDF again after reloading the page.";
    refreshCorrectionOptions();
  });
  // Load a fresh form whenever the proposal opens, unless a draft was restored.
  $effect(() => {
    if (!jobValue.proposing) return;
    if (restored) {
      restored = false;
      return;
    }
    load(jobValue.editing);
  });
  async function manageClient(target: ClientWorkflowTarget = "workspace") {
    if (!correctionJob) return;
    try {
      const href = keepClientWorkflowDraft(
        `job-value:${jobId || "queue"}`,
        pb.authStore.record?.id || "",
        $state.snapshot({
          jobId,
          editing,
          value,
          approver,
          documentFile,
          documentName: documentFile?.name || "",
          correctionJob,
          correctionAllocations,
          originalRate,
          jobUpdated,
          jobReviewToken,
          originalAllocations,
        }),
        correctionJob.client,
        correctionJob.contact || "",
        target,
        (url) => replaceState(url, {}),
      );
      await goto(href);
    } catch (e) {
      error = workflowError(e);
    }
  }
  $effect(() => {
    let current = true;
    approvers = [];
    if (Number.isInteger(value) && Number(value) > 0)
      pb.send(`/api/project_authorization/approvers?value=${value}`, {
        method: "GET",
        requestKey: null,
      })
        .then((response) => {
          if (current) approvers = response.items;
        })
        .catch((e) => {
          if (current) error = workflowError(e);
        });
    return () => {
      current = false;
    };
  });
  async function load(item: WorkflowRecord | null) {
    value = item?.project_value;
    approver = item?.assigned_approver || "";
    documentFile = null;
    error = "";
    correctionLoading = true;
    correctionJob = null;
    correctionStale = false;
    try {
      const id = jobId || item?.job;
      const [context, options] = await Promise.all([
        pb.send(`/api/job_attachments/context?job=${encodeURIComponent(id)}`, { method: "GET" }),
        pb.send("/api/job_requests/options", { method: "GET" }),
      ]);
      const job = context.job as WorkflowRecord;
      const allocations = context.allocations as WorkflowRecord[];
      jobReviewToken = context.review_token;
      correctionJob = { ...job };
      correctionOptions = options;
      correctionAllocations = allocations.map((row) => ({
        division: row.division,
        hours: row.hours,
      }));
      originalRate = job.rate_sheet || "";
      jobUpdated = job.updated;
      originalAllocations = allocationSignature(allocations);
    } catch (e) {
      error = workflowError(e);
    } finally {
      correctionLoading = false;
    }
  }
  async function refreshCorrectionOptions() {
    correctionLoading = true;
    try {
      const [options, context] = await Promise.all([
        pb.send("/api/job_requests/options", { method: "GET", requestKey: null }),
        pb.send(
          `/api/job_attachments/context?job=${encodeURIComponent(jobId || editing?.job || correctionJob?.id || "")}`,
          { method: "GET", requestKey: null },
        ),
      ]);
      correctionOptions = options;
      // Shared client edits change the review digest but must not silently
      // replace a job or its allocations changed by another editor.
      if (
        context.job.updated !== jobUpdated ||
        allocationSignature(context.allocations) !== originalAllocations
      ) {
        correctionStale = true;
        error =
          "The job or its allocations changed while this form was open. Cancel and reopen the value change to review the current job.";
      } else {
        correctionStale = false;
        jobReviewToken = context.review_token;
      }
    } catch (e) {
      error = workflowError(e);
    } finally {
      correctionLoading = false;
    }
  }
  function close() {
    jobValue.proposing = false;
    clearClientWorkflowDraft();
  }
  async function save(event: Event) {
    event.preventDefault();
    if (correctionStale) return;
    busy = true;
    error = "";
    try {
      const body = new FormData();
      body.append("job", jobId || editing?.job || "");
      body.append("project_value", String(value));
      body.append("assigned_approver", approver);
      if (correctionJob) {
        body.append(
          "job_updates",
          JSON.stringify({
            ...Object.fromEntries(
              correctionFields
                .filter((field) => correctionJob![field] !== undefined)
                .map((field) => [field, correctionJob![field]]),
            ),
            authorizing_document: "PA",
          }),
        );
        body.append("job_updated", jobUpdated);
        body.append("job_review_token", jobReviewToken);
        body.append("allocations", JSON.stringify(correctionAllocations));
      }
      // Send the revision loaded with this form so an old form cannot overwrite
      // newer changes to the same attachment ID.
      if (editing) body.append("version", String(editing.version));
      if (documentFile) body.append("document", documentFile);
      await pb.send(`/api/job_attachments${editing ? `/${editing.id}` : ""}`, {
        method: editing ? "PATCH" : "POST",
        body,
      });
      close();
      await jobValue.refresh();
    } catch (e) {
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
</script>

{#if jobValue.proposing}
  <div class="fixed inset-0 z-9999 flex items-start justify-center overflow-y-auto bg-black/50 p-4">
    <form class="my-8 w-full max-w-2xl space-y-3 rounded-lg bg-white p-6 shadow-xl" onsubmit={save}>
      <h2 class="text-xl font-bold">
        {editing ? "Correct value change" : "Propose new project value"}
      </h2>
      <p class="text-sm text-neutral-600">
        The project value stays {valueCAD(effectiveValue)} until the change is approved. Approval covers
        the full new value.
      </p>
      {#if error}<p role="alert" class="rounded bg-red-50 p-3 text-red-800">{error}</p>{/if}
      <label
        >Proposed project value (CAD, before tax)<input
          type="number"
          required
          min="1"
          step="1"
          bind:value
        /></label
      ><label
        >Selected approver<select aria-label="Selected approver" bind:value={approver} required
          ><option value="">Select approver</option>{#each approvers as row (row.id)}<option
              value={row.id}>{row.name}</option
            >{/each}</select
        ></label
      ><label
        >{editing ? "Replacement PDF (optional)" : "Authorization PDF"}<input
          type="file"
          accept="application/pdf,.pdf"
          required={!editing && !documentFile}
          onchange={(event) => (documentFile = event.currentTarget.files?.[0] || null)}
        /></label
      >
      {#if documentFile}<p class="text-sm">Selected PDF: {documentFile.name}</p>{/if}
      {#if correctionLoading}<p>Loading project and billing details…</p>{/if}
      {#if correctionJob && !correctionLoading}<JobValueCorrectionFields
          bind:job={correctionJob}
          bind:allocations={correctionAllocations}
          options={correctionOptions}
          {originalRate}
          onmanageclient={manageClient}
        /><button
          type="button"
          class="text-sm text-blue-700 underline"
          onclick={refreshCorrectionOptions}>Refresh contacts and invoicing profiles</button
        >{/if}
      <div class="flex justify-end gap-2 border-t pt-3">
        <button type="button" disabled={busy} class="rounded border px-4 py-2" onclick={close}
          >Cancel</button
        >
        <button
          disabled={busy || correctionLoading || correctionStale || !correctionJob}
          class="rounded bg-blue-700 px-4 py-2 text-white disabled:opacity-40"
          type="submit">Save proposed value</button
        >
      </div>
    </form>
  </div>
{/if}

<style>
  label {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    font-weight: 600;
    font-size: 0.9rem;
  }
  input,
  select {
    border: 1px solid #a3a3a3;
    padding: 0.5rem;
    border-radius: 0.2rem;
    background: white;
    font-weight: 400;
  }
</style>
