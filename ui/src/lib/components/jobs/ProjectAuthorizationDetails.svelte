<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { shortDate } from "$lib/utilities";
  import Icon from "@iconify/svelte";

  type Person = {
    id?: string;
    given_name?: string;
    surname?: string;
    name?: string;
  };

  type ProjectAuthorizationJob = {
    project_authorization_doc?: string;
    project_authorization_doc_hash?: string;
    project_authorization_doc_url?: string;
    pa_reviewed?: string;
    pa_reviewer?: Person;
    pa_uploaded?: string;
    pa_uploader?: Person;
    pa_rejected?: string;
    pa_rejector?: Person;
    pa_rejection_reason?: string;
  };

  let {
    job,
    status,
    approved,
    rejected,
    canUpload,
    uploading,
    uploadError,
    onUpload,
    canDelete,
    onDelete,
    canRevoke,
    onRevoke,
    canRepairHash,
    onRepairHash,
  }: {
    job: ProjectAuthorizationJob;
    status: string;
    approved: boolean;
    rejected: boolean;
    canUpload: boolean;
    uploading: boolean;
    uploadError: string | null;
    onUpload: (event: Event) => void;
    canDelete: boolean;
    onDelete: () => void;
    canRevoke: boolean;
    onRevoke: () => void;
    canRepairHash: boolean;
    onRepairHash: () => void;
  } = $props();

  function personName(person?: Person) {
    if (!person) return "";
    return `${person.given_name || person.name || ""} ${person.surname || ""}`.trim();
  }
</script>

<div class="flex flex-col gap-2 rounded-sm border border-neutral-200 p-3">
  <div>
    <span class="font-semibold">PA Review:</span>
    {status}
  </div>
  {#if job.pa_uploaded && job.pa_uploader?.id}
    <div>
      <span class="font-semibold">Uploaded By:</span>
      {personName(job.pa_uploader)}
      <span class="text-sm text-neutral-500">
        ({shortDate(job.pa_uploaded, true)})
      </span>
    </div>
  {/if}
  {#if job.project_authorization_doc_url}
    <a
      href={job.project_authorization_doc_url}
      target="_blank"
      rel="noreferrer"
      class="text-blue-600 hover:underline"
    >
      Open PA PDF
    </a>
  {/if}
  {#if job.project_authorization_doc_hash || canRepairHash}
    <div class="flex flex-wrap items-center gap-2">
      <span class="font-semibold">PA Hash:</span>
      {#if canRepairHash}
        <button
          type="button"
          class="font-mono text-sm text-blue-700 underline decoration-dotted underline-offset-2 hover:text-blue-900"
          title="Audit PA document hash"
          onclick={onRepairHash}
        >
          {job.project_authorization_doc_hash
            ? job.project_authorization_doc_hash.slice(0, 8)
            : "No hash"}
        </button>
      {:else if job.project_authorization_doc_hash}
        <span class="font-mono text-sm opacity-70">
          {job.project_authorization_doc_hash.slice(0, 8)}
        </span>
      {/if}
      {#if job.project_authorization_doc_hash}
        <button
          type="button"
          class="text-neutral-500 hover:text-neutral-700"
          title="Copy full PA hash"
          onclick={() => navigator.clipboard.writeText(job.project_authorization_doc_hash ?? "")}
        >
          <Icon icon="mdi:content-copy" width="16" />
        </button>
      {/if}
    </div>
  {/if}
  {#if job.pa_reviewed && job.pa_reviewer?.id}
    <div>
      <span class="font-semibold">Reviewed By:</span>
      {personName(job.pa_reviewer)}
      <span class="text-sm text-neutral-500">
        ({shortDate(job.pa_reviewed, true)})
      </span>
    </div>
  {/if}
  {#if rejected}
    <div class="rounded-sm border border-red-200 bg-red-50 p-2 text-red-900">
      <div>
        <span class="font-semibold">Rejected By Accounting:</span>
        {#if job.pa_rejector?.id}
          {personName(job.pa_rejector)}
        {/if}
        {#if job.pa_rejected}
          <span class="text-sm text-red-700">
            ({shortDate(job.pa_rejected, true)})
          </span>
        {/if}
      </div>
      {#if job.pa_rejection_reason}
        <div class="mt-1 text-sm">{job.pa_rejection_reason}</div>
      {/if}
    </div>
  {/if}
  {#if canUpload && !approved}
    <label class="flex max-w-sm flex-col gap-1 text-sm">
      <span class="font-semibold">Upload Signed PA PDF</span>
      <input
        type="file"
        accept="application/pdf"
        disabled={uploading}
        onchange={onUpload}
        class="rounded-sm border border-neutral-300 p-2"
      />
    </label>
  {/if}
  {#if uploadError}
    <div class="text-sm text-red-600">{uploadError}</div>
  {/if}
  {#if canDelete}
    <div>
      <DsActionButton action={onDelete} color="yellow">Remove PA PDF</DsActionButton>
    </div>
  {/if}
  {#if canRevoke}
    <div>
      <DsActionButton action={onRevoke} color="red">Revoke PA Approval</DsActionButton>
    </div>
  {/if}
</div>
