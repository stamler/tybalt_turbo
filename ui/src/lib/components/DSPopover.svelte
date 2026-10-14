<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    show = $bindable(),
    title,
    subtitle = "",
    error = null,
    submitting = false,
    submitLabel = "Submit",
    danger = false,
    onSubmit,
    onCancel,
    submitDisabled = false,
    children,
  }: {
    show: boolean;
    title: string;
    subtitle?: string;
    error?: string | null;
    submitting?: boolean;
    submitLabel?: string;
    // A destructive action shows a red confirm button.
    danger?: boolean;
    onSubmit: () => void;
    onCancel?: () => void;
    // Keeps the submit button disabled until the form can be submitted.
    submitDisabled?: boolean;
    children: Snippet;
  } = $props();

  function handleCancel() {
    if (onCancel) {
      onCancel();
    } else {
      show = false;
    }
  }
</script>

{#if show}
  <div class="fixed inset-0 z-9999 flex items-center justify-center bg-black/50">
    <div class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
      <h2 class="mb-2 text-xl font-bold">{title}</h2>
      {#if subtitle}
        <p class="mb-4 text-sm text-neutral-600">{subtitle}</p>
      {/if}

      <div class="mb-4 flex flex-col gap-4">
        {@render children()}
      </div>

      {#if error}
        <p class="mb-4 text-sm text-red-600">{error}</p>
      {/if}

      <div class="flex justify-end gap-2">
        <button
          type="button"
          class="rounded-sm bg-neutral-200 px-4 py-2 text-neutral-700 hover:bg-neutral-300"
          onclick={handleCancel}
          disabled={submitting}
        >
          Cancel
        </button>
        <button
          type="button"
          class="rounded-sm px-4 py-2 text-white disabled:opacity-50 {danger
            ? 'bg-red-700 hover:bg-red-800'
            : 'bg-blue-500 hover:bg-blue-600'}"
          onclick={onSubmit}
          disabled={submitting || submitDisabled}
          class:pointer-events-none={submitting || submitDisabled}
        >
          {submitting ? "Saving..." : submitLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
