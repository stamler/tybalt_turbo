<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import type { ClientResponseError } from "pocketbase";
  import { resolve } from "$app/paths";
  import { workflowError, type WorkflowRecord } from "$lib/clientEditors";
  import { JobValue, valueCAD } from "$lib/jobValue.svelte";
  // The form for proposing a new project value or correcting a pending or
  // rejected one. It takes only the value, approver, and PDF; project details
  // are changed in the job editor. It opens as a modal when value.proposing is set.
  let {
    value: jobValue,
    effectiveValue,
    missing = [],
  }: {
    value: JobValue;
    effectiveValue: number;
    // Project details that approval requires and the job lacks.
    missing?: string[];
  } = $props();
  const list = new Intl.ListFormat("en", { type: "conjunction" });
  const jobId = $derived(jobValue.jobId);
  const editing = $derived(jobValue.editing);
  let busy = $state(false),
    error = $state(""),
    // The server named project fields, which are changed in the job editor.
    projectError = $state(false);
  let value = $state<number | undefined>(),
    approver = $state(""),
    documentFile = $state<File | null>(null),
    approvers = $state<WorkflowRecord[]>([]);
  // Start from the change being corrected, or an empty proposal.
  $effect(() => {
    if (!jobValue.proposing) return;
    value = jobValue.editing?.project_value;
    approver = jobValue.editing?.assigned_approver || "";
    documentFile = null;
    error = "";
    projectError = false;
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
  function close() {
    jobValue.proposing = false;
  }
  async function save(event: Event) {
    event.preventDefault();
    if (missing.length) return;
    busy = true;
    error = "";
    projectError = false;
    try {
      const body = new FormData();
      body.append("job", jobId || editing?.job || "");
      body.append("project_value", String(value));
      body.append("assigned_approver", approver);
      // Send the revision loaded with this form so an old form cannot overwrite
      // newer changes to the same attachment ID.
      if (editing) body.append("version", String(editing.version));
      if (documentFile) body.append("document", documentFile);
      const saved = await pb.send(`/api/job_attachments${editing ? `/${editing.id}` : ""}`, {
        method: editing ? "PATCH" : "POST",
        body,
      });
      // A rejected change saved again without a correction stays rejected.
      if (saved.status === "Rejected") {
        error =
          "Nothing has changed since this value change was rejected, so it stays rejected. Correct the project in the job editor, or change the value, approver, or PDF.";
        projectError = true;
        return;
      }
      close();
      await jobValue.refresh();
    } catch (e) {
      error = workflowError(e);
      projectError = Object.keys((e as ClientResponseError).response?.data ?? {}).length > 0;
    } finally {
      busy = false;
    }
  }
</script>

{#if jobValue.proposing}
  <div class="fixed inset-0 z-9999 flex items-start justify-center overflow-y-auto bg-black/50 p-4">
    <form class="my-8 w-full max-w-lg space-y-3 rounded-lg bg-white p-6 shadow-xl" onsubmit={save}>
      <h2 class="text-xl font-bold">
        {editing ? "Correct value change" : "Propose new project value"}
      </h2>
      <p class="text-sm text-neutral-600">
        The project value stays {valueCAD(effectiveValue)} until the change is approved. Approval covers
        the full new value.
      </p>
      {#if missing.length}
        <div role="alert" class="rounded border border-amber-300 bg-amber-50 p-3 text-sm">
          <p>Approval needs {list.format(missing)} on this project.</p>
          <a class="text-blue-700 underline" href={resolve(`/jobs/${jobId}/edit`)}
            >Edit the project</a
          >
        </div>
      {/if}
      {#if error}<div role="alert" class="rounded bg-red-50 p-3 text-sm text-red-800">
          <p>{error}</p>
          {#if projectError}<a class="text-blue-700 underline" href={resolve(`/jobs/${jobId}/edit`)}
              >Edit the project</a
            >{/if}
        </div>{/if}
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
      <div class="flex justify-end gap-2 border-t pt-3">
        <button type="button" disabled={busy} class="rounded border px-4 py-2" onclick={close}
          >Cancel</button
        >
        <button
          disabled={busy || missing.length > 0}
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
