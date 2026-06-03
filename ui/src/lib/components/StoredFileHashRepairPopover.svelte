<script lang="ts">
  import Icon from "@iconify/svelte";
  import { pb } from "$lib/pocketbase";

  export let show = false;
  export let recordId = "";
  export let title = "Attachment Repair";
  export let auditPath = "";
  export let replacePath = "";
  export let markMissingPath = "";
  export let canMarkMissing = true;
  export let hasAttachment = false;
  export let currentHash = "";
  export let currentUpdated = "";
  export let missingReason = "";
  export let currentHashLabel = "Current Hash";
  export let markMissingLabel = "Mark attachment missing";
  export let markMissingPlaceholder =
    "Explain why this historical attachment is missing and cannot be recovered.";
  export let auditMatchesMessage = "Stored hash matches the attachment.";
  export let auditMismatchMessage = "Stored hash does not match the attachment.";
  export let replaceConfirmMessage =
    "Replacing this stored hash is irreversible. Verify that the attachment opens and is actually usable before accepting the calculated hash.";
  export let replaceNoopMessage = "No change made. The stored hash already matches the attachment.";
  export let replaceSuccessMessage = "Stored hash replaced with the calculated attachment hash.";
  export let onClose: () => void = () => {};
  export let onRepaired: () => Promise<void> | void = () => {};

  type AuditResponse = {
    expense_id?: string;
    job_id?: string;
    target_collection: string;
    target_id: string;
    filename: string;
    storage_path: string;
    stored_hash: string;
    updated: string;
    calculated_hash: string;
    matches: boolean;
  };

  type ReplaceResponse = AuditResponse & {
    previous_hash: string;
    new_hash: string;
    replaced: boolean;
    noop: boolean;
  };

  type MarkMissingResponse = {
    expense_id: string;
    updated: string;
    attachment_missing_reason: string;
    previous_attachment_document: string;
    marked: boolean;
    noop: boolean;
  };

  type MessageTone = "green" | "red" | "orange" | "neutral";

  let auditResult: AuditResponse | null = null;
  let missingReasonDraft = "";
  let message = "";
  let messageTone: MessageTone = "neutral";
  let loading = false;

  $: displayId = recordId;

  $: if (!show) {
    auditResult = null;
    missingReasonDraft = "";
    message = "";
    messageTone = "neutral";
    loading = false;
  }

  $: if (show && !missingReasonDraft) {
    missingReasonDraft = missingReason;
  }

  function close() {
    onClose();
  }

  function messageClass(tone: MessageTone) {
    if (tone === "green") return "border-green-300 bg-green-50 text-green-800";
    if (tone === "red") return "border-red-300 bg-red-50 text-red-800";
    if (tone === "orange") return "border-orange-300 bg-orange-50 text-orange-800";
    return "border-neutral-300 bg-neutral-50 text-neutral-700";
  }

  function errorMessage(error: any, fallback: string) {
    return error?.response?.message || error?.response?.error || fallback;
  }

  async function auditHash() {
    if (!auditPath) return;
    loading = true;
    try {
      const result = (await pb.send(auditPath, {
        method: "POST",
      })) as AuditResponse;
      auditResult = result;
      if (result.matches) {
        message = auditMatchesMessage;
        messageTone = "green";
      } else {
        message = auditMismatchMessage;
        messageTone = "red";
      }
    } catch (error: any) {
      message = errorMessage(error, "Hash audit failed");
      messageTone = "red";
    } finally {
      loading = false;
    }
  }

  async function replaceHash() {
    if (!auditResult || !replacePath) return;
    const confirmed = window.confirm(replaceConfirmMessage);
    if (!confirmed) return;

    loading = true;
    try {
      const result = (await pb.send(replacePath, {
        method: "POST",
        body: { updated: auditResult.updated },
      })) as ReplaceResponse;
      auditResult = result;
      if (result.noop) {
        message = replaceNoopMessage;
        messageTone = "neutral";
      } else {
        message = replaceSuccessMessage;
        messageTone = "orange";
        await onRepaired();
      }
    } catch (error: any) {
      message = errorMessage(error, "Hash replacement failed");
      messageTone = "red";
    } finally {
      loading = false;
    }
  }

  async function markMissing() {
    const reason = missingReasonDraft.trim();
    if (!reason || !currentUpdated || !markMissingPath) return;

    const confirmed = window.confirm(
      "Marking this attachment missing is irreversible. Use this only after verifying that the original receipt cannot be recovered.",
    );
    if (!confirmed) return;

    loading = true;
    try {
      const result = (await pb.send(markMissingPath, {
        method: "POST",
        body: { updated: currentUpdated, reason },
      })) as MarkMissingResponse;
      if (result.noop) {
        message = "No change made. The attachment was already marked missing with this reason.";
        messageTone = "neutral";
      } else {
        message = "Attachment marked missing.";
        messageTone = "orange";
        await onRepaired();
      }
    } catch (error: any) {
      message = errorMessage(error, "Marking attachment missing failed");
      messageTone = "red";
    } finally {
      loading = false;
    }
  }
</script>

{#if show}
  <div class="fixed inset-0 z-9999 flex items-center justify-center bg-black/50 px-4">
    <div class="w-full max-w-2xl rounded-lg bg-white p-5 shadow-xl">
      <div class="mb-4 flex items-start justify-between gap-4">
        <div>
          <h2 class="text-xl font-bold">{title}</h2>
          <p class="mt-1 text-sm text-neutral-600">{displayId}</p>
        </div>
        <button
          type="button"
          class="rounded-sm p-1 text-neutral-500 hover:bg-neutral-100 hover:text-neutral-800"
          title="Close"
          on:click={close}
          disabled={loading}
        >
          <Icon icon="mdi:close" width="22" />
        </button>
      </div>

      <div class="space-y-3 text-sm">
        <div>
          <div class="font-semibold text-neutral-700">{currentHashLabel}</div>
          <div class="break-all font-mono text-neutral-700">{currentHash || "No stored hash"}</div>
        </div>

        {#if missingReason}
          <div>
            <div class="font-semibold text-neutral-700">Missing Reason</div>
            <div class="rounded-sm border border-amber-300 bg-amber-50 px-3 py-2 text-amber-900">
              {missingReason}
            </div>
          </div>
        {/if}

        {#if auditResult}
          <div class="grid gap-3 md:grid-cols-2">
            <div>
              <div class="font-semibold text-neutral-700">Target</div>
              <div class="font-mono text-neutral-700">{auditResult.target_collection}/{auditResult.target_id}</div>
            </div>
            <div>
              <div class="font-semibold text-neutral-700">File</div>
              <div class="break-all text-neutral-700">{auditResult.filename}</div>
            </div>
          </div>
          <div>
            <div class="font-semibold text-neutral-700">Calculated</div>
            <div class="break-all font-mono text-neutral-700">{auditResult.calculated_hash}</div>
          </div>
        {/if}

        {#if message}
          <div class="rounded-sm border px-3 py-2 {messageClass(messageTone)}">{message}</div>
        {/if}

        {#if canMarkMissing}
          <div class="border-t pt-3">
            <label class="block text-sm font-semibold text-neutral-700" for="missing-reason">
              {markMissingLabel}
            </label>
            <textarea
              id="missing-reason"
              class="mt-1 min-h-24 w-full rounded-sm border border-neutral-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-hidden"
              placeholder={markMissingPlaceholder}
              bind:value={missingReasonDraft}
              disabled={loading}
            ></textarea>
          </div>
        {/if}
      </div>

      <div class="mt-5 flex flex-wrap justify-end gap-2">
        <button
          type="button"
          class="rounded-sm bg-neutral-200 px-4 py-2 text-neutral-700 hover:bg-neutral-300 disabled:opacity-50"
          on:click={auditHash}
          disabled={loading || !hasAttachment || !auditPath}
        >
          {loading ? "Working..." : "Audit"}
        </button>
        <button
          type="button"
          class="rounded-sm bg-orange-500 px-4 py-2 text-white hover:bg-orange-600 disabled:opacity-50"
          on:click={replaceHash}
          disabled={loading || !hasAttachment || !auditResult || !replacePath}
        >
          Replace
        </button>
        {#if canMarkMissing}
          <button
            type="button"
            class="rounded-sm bg-red-600 px-4 py-2 text-white hover:bg-red-700 disabled:opacity-50"
            on:click={markMissing}
            disabled={loading || !missingReasonDraft.trim() || !currentUpdated || !markMissingPath}
          >
            Mark Missing
          </button>
        {/if}
      </div>
    </div>
  </div>
{/if}
