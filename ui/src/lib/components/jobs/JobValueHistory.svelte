<script lang="ts">
  import JobValueCorrectionFields from "./JobValueCorrectionFields.svelte";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import { goto, invalidateAll, replaceState } from "$app/navigation";
  import { onMount } from "svelte";
  import {
    clearClientWorkflowDraft,
    type ClientWorkflowTarget,
    keepClientWorkflowDraft,
    readClientWorkflowDraft,
  } from "$lib/clientWorkflowNavigation";
  import { wholeCAD } from "$lib/jobRequests";
  import { workflowError, type WorkflowRecord } from "$lib/clientEditors";
  let {
    jobId,
    effectiveValue,
    queue = false,
  }: { jobId?: string; effectiveValue?: number; queue?: boolean } = $props();
  let items = $state<WorkflowRecord[]>([]),
    busy = $state(false),
    error = $state(""),
    loading = $state(true),
    editing = $state<WorkflowRecord | null>(null),
    creating = $state(false);
  let value = $state<number | undefined>(),
    approver = $state(""),
    documentFile = $state<File | null>(null),
    approvers = $state<WorkflowRecord[]>([]);
  let rejectionReasons = $state<Record<string, string>>({});
  let correctionJob = $state<WorkflowRecord | null>(null),
    correctionOptions = $state<WorkflowRecord>({}),
    correctionAllocations = $state<WorkflowRecord[]>([]),
    correctionLoading = $state(false),
    originalRate = $state(""),
    jobUpdated = $state(""),
    jobReviewToken = $state(""),
    originalAllocations = $state(""),
    correctionStale = $state(false);
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
  let confirmedReview = $state<Record<string, boolean>>({});
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
  onMount(() => {
    const draft = readClientWorkflowDraft<ValueDraft>(
      `job-value:${jobId || "queue"}`,
      pb.authStore.record?.id || "",
    );
    if (!draft || draft.jobId !== jobId) return;
    editing = draft.editing;
    value = draft.value;
    approver = draft.approver;
    documentFile = draft.documentFile instanceof File ? draft.documentFile : null;
    correctionJob = draft.correctionJob;
    correctionAllocations = draft.correctionAllocations;
    originalRate = draft.originalRate;
    jobUpdated = draft.jobUpdated;
    jobReviewToken = draft.jobReviewToken;
    originalAllocations = draft.originalAllocations;
    creating = true;
    if (draft.documentName && !documentFile)
      error = "Select the PDF again after reloading the page.";
    refreshCorrectionOptions();
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
  const canEdit = $derived($globalStore.claims.includes("job")),
    admin = $derived($globalStore.claims.includes("admin"));
  const pending = $derived(items.find((item) => item.status !== "Approved"));
  // Sequence orders approved entries within this job, so only the latest later
  // approval can offer value restoration. The server enforces the same rule.
  const latest = $derived(
    Math.max(
      0,
      ...items.filter((item) => item.status === "Approved").map((item) => item.approval_sequence),
    ),
  );
  const money = (amount: number) =>
    new Intl.NumberFormat("en-CA", {
      style: "currency",
      currency: "CAD",
      maximumFractionDigits: 0,
    }).format(amount);
  $effect(() => {
    jobId;
    queue;
    reload();
  });
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
  async function reload() {
    loading = true;
    try {
      const result = await pb.send(
        `/api/job_attachments?${queue ? "queue=approval" : `job=${jobId}`}`,
        { method: "GET", requestKey: null },
      );
      items = result.items;
      confirmedReview = {};
    } catch (e) {
      error = workflowError(e);
    } finally {
      loading = false;
    }
  }
  async function edit(item: WorkflowRecord | null) {
    editing = item;
    creating = true;
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
  async function refresh() {
    await reload();
    await Promise.all([globalStore.refreshAttentionCounts(), invalidateAll()]);
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
      creating = false;
      clearClientWorkflowDraft();
      await refresh();
    } catch (e) {
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
  async function action(item: WorkflowRecord, name: string) {
    if (
      name === "delete" &&
      !window.confirm(
        item.status === "Approved"
          ? `Delete this approval and restore the previous project value of ${money(item.previous_project_value)}? The PDF and history will be deleted.`
          : "Delete this pending value change and its PDF?",
      )
    )
      return;
    busy = true;
    error = "";
    try {
      await pb.send(`/api/job_attachments/${item.id}${name === "delete" ? "" : `/${name}`}`, {
        method: name === "delete" ? "DELETE" : "POST",
        body: {
          // Use the displayed revision for approval, rejection, and deletion.
          // A decision can change it even when the PDF hash stays the same.
          version: item.version,
          document_hash: item.document_hash,
          // A rejection records the job data shown here.
          review_token: item.current_review_token,
          rejection_reason: rejectionReasons[item.id] ?? "",
        },
      });
      delete rejectionReasons[item.id];
      await refresh();
    } catch (e) {
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
  async function openPDF(item: WorkflowRecord) {
    error = "";
    try {
      const response = await fetch(`${pb.baseURL}/api/job_attachments/${item.id}/document`, {
        headers: { Authorization: pb.authStore.token },
      });
      if (!response.ok) throw new Error("Cannot load this PDF.");
      const url = URL.createObjectURL(await response.blob());
      window.open(url, "_blank");
      setTimeout(() => URL.revokeObjectURL(url), 60000);
    } catch (e) {
      error = workflowError(e);
    }
  }
</script>

<section class="space-y-4 rounded border border-neutral-300 p-4">
  <h2 class="text-lg font-semibold">
    {queue ? "Pending project-value approvals" : "Project value and approval history"}
  </h2>
  {#if effectiveValue !== undefined}<p class="text-lg">
      Effective project value: <strong>{money(effectiveValue)}</strong> CAD, before tax
    </p>{/if}
  {#if error}<p role="alert" class="rounded bg-red-50 p-3 text-red-800">{error}</p>{/if}
  {#if loading}<p>Loading approval history…</p>{/if}
  {#if !queue && canEdit && !pending && !creating}<button
      class="rounded bg-blue-700 px-4 py-2 text-white"
      onclick={() => edit(null)}>Propose a new project value</button
    >{/if}
  {#if creating}<form class="space-y-3 rounded bg-neutral-50 p-4" onsubmit={save}>
      <h3 class="font-semibold">
        {editing ? "Correct pending value change" : "New project-value change"}
      </h3>
      <p class="text-sm">
        The effective job value stays unchanged until approval. Approval covers the full new value.
      </p>
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
          ><option value="">Select approver</option>{#each approvers as row}<option value={row.id}
              >{row.name} — up to {money(row.max_value)}</option
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
      <button
        disabled={busy || correctionLoading || correctionStale || !correctionJob}
        class="rounded bg-blue-700 px-4 py-2 text-white"
        type="submit">Save proposed value</button
      >
      <button
        type="button"
        disabled={busy}
        class="rounded border px-4 py-2"
        onclick={() => {
          creating = false;
          clearClientWorkflowDraft();
        }}>Cancel</button
      >
    </form>{/if}
  {#each items as item}<article class="space-y-2 border-t py-3">
      <div class="flex flex-wrap justify-between gap-2">
        <h3 class="font-semibold">
          {item.status} · {wholeCAD(item.project_value)}{item.initial_approval
            ? " · Initial approval"
            : ""}
        </h3>
        <button onclick={() => openPDF(item)} class="text-blue-700 underline">Open PDF</button>
      </div>
      {#if queue}<p>
          Effective project value: <strong>{money(item.effective_project_value)}</strong> CAD, before
          tax
        </p>
        <p>
          Client: {item.related?.client?.name || "Not provided"} · Manager: {item.related?.manager
            ?.name || "Not provided"}
        </p>
        <a href={`/jobs/${item.job}/details`} class="text-blue-700 underline"
          >{item.job_number || item.job} — {item.job_description || "Job details"}</a
        >{/if}
      {#if item.status === "Approved"}<p>
          Approved by {item.approver_name || item.approver} on {item.approved}. Recorded limit: {money(
            item.approval_max_value,
          )}.
        </p>
        <p class="text-sm">
          Approval sequence: {item.approval_sequence}{!item.initial_approval
            ? ` · Previous value: ${money(item.previous_project_value)}`
            : ""}
        </p>
        {#if item.checklist_version}<details>
            <summary>Approval checklist version {item.checklist_version}</summary
            >{#each Object.entries(item.checklist_answers || {}) as [key, checked]}<p>
                {key.replaceAll("_", " ")}: {checked ? "Confirmed" : "Not confirmed"}
              </p>{/each}
          </details>{/if}{:else}<p>
          Selected approver: {item.assigned_approver_name || item.assigned_approver}
        </p>{/if}
      {#if item.rejection_reason}<p class="text-red-800">
          Rejection reason: {item.rejection_reason}
        </p>{/if}
      {#if canEdit && item.status !== "Approved" && !queue}<button
          class="rounded border px-3 py-2"
          onclick={() => edit(item)}>Correct value change</button
        >{/if}
      {#if item.assigned_approver === pb.authStore.record?.id && $globalStore.claims.includes("project_authorization") && item.status === "Pending"}<div
          class="space-y-2"
        >
          <label class="review"
            ><input type="checkbox" bind:checked={confirmedReview[item.id]} />I reviewed this PDF
            and its full proposed value.</label
          ><button
            disabled={busy || !confirmedReview[item.id]}
            class="rounded bg-green-700 px-3 py-2 text-white disabled:opacity-40"
            onclick={() => action(item, "approve")}>Approve value</button
          ><label
            >Rejection reason<textarea bind:value={rejectionReasons[item.id]}></textarea></label
          ><button
            disabled={busy || !rejectionReasons[item.id]?.trim()}
            class="rounded border border-red-700 px-3 py-2 text-red-800 disabled:opacity-40"
            onclick={() => action(item, "reject")}>Reject</button
          >
        </div>{/if}
      {#if admin && !item.initial_approval && (item.status !== "Approved" || (!pending && item.approval_sequence === latest))}<button
          disabled={busy}
          class="rounded border border-red-700 px-3 py-2 text-red-800"
          onclick={() => action(item, "delete")}
          >Delete {item.status === "Approved"
            ? "latest approval and restore value"
            : "pending change"}</button
        >{/if}
    </article>{:else}{#if !loading}<p>No value approval records.</p>{/if}{/each}
</section>

<style>
  label {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    font-weight: 600;
    font-size: 0.9rem;
  }
  input,
  select,
  textarea {
    border: 1px solid #a3a3a3;
    padding: 0.5rem;
    border-radius: 0.2rem;
    background: white;
    font-weight: 400;
  }
  .review {
    flex-direction: row;
    align-items: center;
  }
</style>
