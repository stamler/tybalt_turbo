<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { goto, invalidateAll } from "$app/navigation";
  import { globalStore } from "$lib/stores/global";
  import { jobRequestsEnabled } from "$lib/stores/appConfig";
  import RequestEditor from "$lib/components/jobs/RequestEditor.svelte";
  import {
    emptyChecklist,
    requestChecklist,
    contactEmail,
    fullName,
    wholeCAD,
  } from "$lib/jobRequests";
  import { workflowError, type WorkflowRecord } from "$lib/clientEditors";
  import { clientLabel } from "$lib/clientWorkspace";
  let { data } = $props();
  let busy = $state(false),
    error = $state(""),
    dirty = $state(false),
    printing = $state(false),
    // Notes may be internal, so the printout leaves them out unless chosen.
    printNotes = $state(false);
  let checklist = $state(emptyChecklist()),
    duplicateChecked = $state(false),
    rejectionReason = $state("");
  let editor: RequestEditor | undefined = $state();
  const detail = $derived(data.detail),
    request = $derived(detail.request),
    permission = $derived(detail.permissions),
    related = $derived(detail.related);
  const allChecked = $derived(Object.values(checklist).every(Boolean));
  $effect(() => {
    // The server digest includes submission_token. Resubmission changes it even
    // when the request version/PDF are unchanged; require six fresh answers.
    detail.review_token;
    request.version;
    checklist = emptyChecklist();
    duplicateChecked = false;
  });
  async function refresh() {
    await Promise.all([invalidateAll(), globalStore.refreshAttentionCounts()]);
  }
  async function action(name: string) {
    busy = true;
    error = "";
    try {
      const result = await pb.send(
        `/api/job_requests/${request.id}${name === "delete" ? "" : `/${name}`}`,
        {
          method: name === "delete" ? "DELETE" : "POST",
          body: {
            version: request.version,
            // Echo the reviewed digest, never reconstruct it from current data.
            // It binds this page's checklist to the server's submission_token;
            // an old page must fail after recall and resubmission.
            review_token: detail.review_token,
            document_hash: detail.attachment?.document_hash,
            checklist_version: 1,
            checklist_answers: checklist,
            rejection_reason: rejectionReason,
            duplicate_work_checked: duplicateChecked,
          },
        },
      );
      if (name === "create") {
        await globalStore.refreshAttentionCounts();
        await goto(`/jobs/${result.job?.id ?? result.id ?? result.job_id}/details`);
      } else if (name === "delete") {
        await globalStore.refreshAttentionCounts();
        await goto("/job-requests?queue=abandoned");
      } else await refresh();
    } catch (e) {
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
  async function upload(event: Event) {
    const input = event.currentTarget as HTMLInputElement,
      file = input.files?.[0];
    if (!file) return;
    busy = true;
    error = "";
    try {
      const body = new FormData();
      body.append("document", file);
      body.append("version", String(request.version));
      await pb.send(`/api/job_requests/${request.id}/document`, { method: "POST", body });
      await refresh();
    } catch (e) {
      error = workflowError(e);
    } finally {
      busy = false;
      input.value = "";
    }
  }
  async function openPDF() {
    busy = true;
    error = "";
    try {
      const response = await fetch(
        `${pb.baseURL}/api/job_attachments/${detail.attachment?.id}/document`,
        { headers: { Authorization: pb.authStore.token } },
      );
      if (!response.ok) throw new Error("Cannot load the authorization PDF.");
      const url = URL.createObjectURL(await response.blob());
      window.open(url, "_blank");
      setTimeout(() => URL.revokeObjectURL(url), 60000);
    } catch (e) {
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
  async function print(saved = false) {
    if (dirty && !saved) {
      printing = true;
      return;
    }
    if (saved && !(await editor?.save())) return;
    printing = false;
    await goto(`/job-requests/${request.id}/print${printNotes ? "?notes=1" : ""}`);
  }
</script>

<div class="mx-auto max-w-5xl space-y-5 p-4">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-2xl font-bold">Project Request</h1>
      <p class="text-sm text-neutral-600">
        {request.id} · Version {request.version} · {request.status || request.state}
      </p>
    </div>
    <div class="flex flex-wrap items-center gap-3">
      {#if request.notes}<label class="flex items-center gap-2 text-sm"
          ><input type="checkbox" bind:checked={printNotes} />Include notes</label
        >{/if}
      <button class="rounded border px-4 py-2" onclick={() => print()}
        >Print Project Authorization</button
      >
    </div>
  </div>
  <p class="text-sm text-neutral-600">
    Requested by {fullName(related.creator) || request.creator}. Selected approver: {fullName(
      related.approver,
    ) || "Not selected"}.
  </p>
  {#if error}<p class="rounded border border-red-300 bg-red-50 p-3 text-red-800" role="alert">
      {error}
    </p>{/if}
  {#if printing}<div class="space-y-3 rounded border border-amber-400 p-4" role="alert">
      <p>Save your changes before printing.</p>
      <button class="rounded bg-blue-700 px-3 py-2 text-white" onclick={() => print(true)}
        >Save and print</button
      >
      <button class="rounded border px-3 py-2" onclick={() => (printing = false)}
        >Cancel printing</button
      >
    </div>{/if}
  {#if !$jobRequestsEnabled}<p class="rounded bg-amber-50 p-3">
      Request processing is disabled. Printing remains available.
    </p>{/if}
  {#if request.rejection_reason}<p class="rounded bg-red-50 p-3">
      Rejection reason: {request.rejection_reason}. The creator must make a correction before
      submission.
    </p>{/if}
  {#if permission.can_edit && $jobRequestsEnabled}
    {#key request.version}<RequestEditor
        bind:this={editor}
        {request}
        allocations={detail.allocations}
        options={data.options}
        bind:dirty
        onsaved={refresh}
      />{/key}
  {:else}
    <section class="space-y-2 rounded bg-neutral-100 p-4">
      <h2 class="text-lg font-semibold whitespace-pre-wrap">
        {request.description || "Untitled request"}
      </h2>
      <p>Client: {related.client ? clientLabel(related.client) : "Not provided"}</p>
      <p>
        Project contact: {fullName(related.contact) || "Not provided"} · {contactEmail(
          related.contact,
        ) || "Not provided"}
      </p>
      <p>
        Manager: {fullName(related.manager) || "Not provided"} · Branch: {related.branch?.name ||
          "Not provided"}
      </p>
      {#if request.notes}<p class="whitespace-pre-wrap">Notes: {request.notes}</p>{/if}
      <p>Location: {request.location || "Not provided"}</p>
      <p>
        Project value: {wholeCAD(request.project_value)} · {request.time_and_materials_set
          ? request.time_and_materials
            ? "Time and materials"
            : "Fixed fee"
          : "Fee basis not provided"}
      </p>
      <p>
        Rate sheet: {related.rate_sheet?.name || "Not provided"} · Revision {related.rate_sheet
          ?.revision ?? "Not provided"}
      </p>
      <p>First Nations agreement: {request.fn_agreement_answer || "Not answered"}</p>
      <p>Intended project status: {request.project_status}</p>
      <p>
        Award date: {request.project_award_date || "Not provided"} · Completion date: {request.project_completion_date ||
          "Not provided"}
      </p>
      <p>Bill to: {related.bill_to_name || "Not provided"}</p>
      <p>
        Billing contact: {fullName(related.billing_contact) || "Not provided"} · {contactEmail(
          related.billing_contact,
        ) || "Not provided"}
      </p>
      <p class="whitespace-pre-wrap">
        Invoicing instructions: {related.invoicing_information?.invoicing_instructions || "None"}
      </p>
      {#if related.parent}<p>
          Parent: <a href={`/jobs/${related.parent.id}/details`} class="text-blue-700 underline"
            >{related.parent.number} — {related.parent.description}</a
          >
        </p>{/if}
      {#if related.proposal}<p>
          Proposal: <a href={`/jobs/${related.proposal.id}/details`} class="text-blue-700 underline"
            >{related.proposal.number} — {related.proposal.description}</a
          >
        </p>{/if}
      {#if request.client_po}<p>
          Client PO: {request.client_po}
        </p>{/if}{#if request.client_reference_number}<p>
          Client reference: {request.client_reference_number}
        </p>{/if}
      <h3 class="font-semibold">Division allocations</h3>
      {#each detail.allocations as allocation}<p>
          {allocation.division_name || allocation.expand?.division?.name || allocation.division}: {allocation.hours}
          h
        </p>{/each}
    </section>
  {/if}
  <section class="space-y-3 rounded border p-4">
    <h2 class="font-semibold">Authorization evidence</h2>
    {#if detail.attachment}<button class="text-blue-700 underline" disabled={busy} onclick={openPDF}
        >Open authorization PDF</button
      >
      <p class="text-xs break-all text-neutral-500">
        SHA-256: {detail.attachment.document_hash}
      </p>{:else}<p>No authorization PDF uploaded.</p>{/if}
    {#if permission.can_edit && $jobRequestsEnabled}<label class="block text-sm font-semibold"
        >{detail.attachment ? "Replace PDF" : "Upload PDF"}<input
          class="mt-2 block"
          type="file"
          accept="application/pdf,.pdf"
          disabled={busy || dirty}
          onchange={upload}
        /></label
      >
      <p class="text-sm text-neutral-600">
        Save edits first. Upload the signed copy, including the rate appendix for T&amp;M work, or
        other sufficient authorization evidence.
      </p>{/if}
  </section>
  {#if permission.can_approve || permission.can_create}
    <section class="space-y-3 rounded border p-4">
      <h2 class="font-semibold">Duplicate-work review</h2>
      <p class="text-sm">
        Possible matches require review. A match alone does not prove that the work is duplicated.
      </p>
      {#each data.duplicates.jobs || [] as job}<div class="border-t py-2">
          <a href={`/jobs/${job.id}/details`} class="font-semibold text-blue-700 underline"
            >{job.number} — {job.description}</a
          >
          <p>
            {job.client_name} · {job.status} · {job.manager_name}
            {job.parent_number ? `· Parent ${job.parent_number}` : ""}
          </p>
          <p class="text-sm text-neutral-600">{(job.reasons || []).join("; ")}</p>
        </div>{:else}<p>No possible matching jobs found.</p>{/each}
      {#if permission.can_create}<h3 class="font-semibold">Other pending requests</h3>
        {#each data.duplicates.requests || [] as other}<div>
            <a class="text-blue-700 underline" href={`/job-requests/${other.id}`}
              >{other.id} — {other.description}</a
            >
            <p>{(other.reasons || []).join("; ")}</p>
          </div>{:else}<p>No possible matching requests found.</p>{/each}{/if}
    </section>
  {/if}
  {#if $jobRequestsEnabled}<section class="space-y-4 rounded border p-4">
      <h2 class="font-semibold">Workflow actions</h2>
      {#if permission.can_approve}<div class="space-y-3">
          {#each Object.entries(requestChecklist) as [key, label]}<label
              class="flex items-start gap-2"
              ><input type="checkbox" bind:checked={checklist[key]} class="mt-1" />{label}</label
            >{/each}<button
            disabled={busy || dirty || !allChecked}
            class="rounded bg-green-700 px-4 py-2 text-white disabled:opacity-40"
            onclick={() => action("approve")}>Approve request</button
          >
        </div>{/if}
      {#if permission.can_create}<label class="flex gap-2"
          ><input type="checkbox" bind:checked={duplicateChecked} />I checked existing jobs and
          other pending requests. This job does not duplicate existing work.</label
        ><button
          disabled={busy || !duplicateChecked}
          class="rounded bg-green-700 px-4 py-2 text-white disabled:opacity-40"
          onclick={() => action("create")}>Create Job</button
        >{/if}
      {#if permission.can_submit}<button
          disabled={busy || dirty}
          class="rounded bg-blue-700 px-4 py-2 text-white disabled:opacity-40"
          onclick={() => action("submit")}>Submit</button
        >{/if}
      {#if permission.can_recall}<button
          disabled={busy}
          class="rounded border px-4 py-2"
          onclick={() => action("recall")}>Recall for editing</button
        >{/if}
      {#if permission.can_reject}<div class="space-y-2">
          <label class="block font-semibold" for="reason">Rejection reason</label><textarea
            id="reason"
            class="w-full rounded border p-2"
            bind:value={rejectionReason}></textarea><button
            disabled={busy || !rejectionReason.trim()}
            class="rounded border border-red-700 px-4 py-2 text-red-800 disabled:opacity-40"
            onclick={() => action("reject")}>Reject</button
          >
        </div>{/if}
    </section>{/if}
  {#if permission.can_delete}<button
      disabled={busy}
      class="rounded border border-red-700 px-4 py-2 text-red-800"
      onclick={() => {
        if (
          window.confirm(
            "Delete this abandoned request, its PDF, and its allocations? This cannot be undone.",
          )
        )
          action("delete");
      }}>Delete abandoned request</button
    >{/if}
</div>
