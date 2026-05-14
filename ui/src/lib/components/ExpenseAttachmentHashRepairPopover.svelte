<script lang="ts">
  import Icon from "@iconify/svelte";
  import { pb } from "$lib/pocketbase";

  export let show = false;
  export let expenseId: string;
  export let currentHash = "";
  export let onClose: () => void = () => {};
  export let onRepaired: () => Promise<void> | void = () => {};

  type AuditResponse = {
    expense_id: string;
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

  type MessageTone = "green" | "red" | "orange" | "neutral";

  let auditResult: AuditResponse | null = null;
  let message = "";
  let messageTone: MessageTone = "neutral";
  let loading = false;

  $: if (!show) {
    auditResult = null;
    message = "";
    messageTone = "neutral";
    loading = false;
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
    loading = true;
    try {
      const result = (await pb.send(`/api/expenses/${expenseId}/attachment_hash/audit`, {
        method: "POST",
      })) as AuditResponse;
      auditResult = result;
      if (result.matches) {
        message = "Stored hash matches the attachment.";
        messageTone = "green";
      } else {
        message = "Stored hash does not match the attachment.";
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
    if (!auditResult) return;
    const confirmed = window.confirm(
      "Replacing this stored hash is irreversible. Verify that the attachment opens and is actually usable before accepting the calculated hash.",
    );
    if (!confirmed) return;

    loading = true;
    try {
      const result = (await pb.send(`/api/expenses/${expenseId}/attachment_hash/replace`, {
        method: "POST",
        body: { updated: auditResult.updated },
      })) as ReplaceResponse;
      auditResult = result;
      if (result.noop) {
        message = "No change made. The stored hash already matches the attachment.";
        messageTone = "neutral";
      } else {
        message = "Stored hash replaced with the calculated attachment hash.";
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
</script>

{#if show}
  <div class="fixed inset-0 z-9999 flex items-center justify-center bg-black/50 px-4">
    <div class="w-full max-w-2xl rounded-lg bg-white p-5 shadow-xl">
      <div class="mb-4 flex items-start justify-between gap-4">
        <div>
          <h2 class="text-xl font-bold">Attachment Hash</h2>
          <p class="mt-1 text-sm text-neutral-600">{expenseId}</p>
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
          <div class="font-semibold text-neutral-700">Current</div>
          <div class="break-all font-mono text-neutral-700">{currentHash || "No stored hash"}</div>
        </div>

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
      </div>

      <div class="mt-5 flex justify-end gap-2">
        <button
          type="button"
          class="rounded-sm bg-neutral-200 px-4 py-2 text-neutral-700 hover:bg-neutral-300 disabled:opacity-50"
          on:click={auditHash}
          disabled={loading}
        >
          {loading ? "Working..." : "Audit"}
        </button>
        <button
          type="button"
          class="rounded-sm bg-orange-500 px-4 py-2 text-white hover:bg-orange-600 disabled:opacity-50"
          on:click={replaceHash}
          disabled={loading || !auditResult}
        >
          Replace
        </button>
      </div>
    </div>
  </div>
{/if}
