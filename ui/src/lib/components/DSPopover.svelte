<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    show = $bindable(),
    title,
    subtitle = "",
    error = null,
    submitting = false,
    submitLabel = "Submit",
    onSubmit,
    onCancel,
    children,
  }: {
    show: boolean;
    title: string;
    subtitle?: string;
    error?: string | null;
    submitting?: boolean;
    submitLabel?: string;
    onSubmit: () => void;
    onCancel?: () => void;
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
  <div class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/50">
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
          class="rounded bg-neutral-200 px-4 py-2 text-neutral-700 hover:bg-neutral-300"
          onclick={handleCancel}
          disabled={submitting}
        >
          Cancel
        </button>
        <button
          type="button"
          class="rounded bg-blue-500 px-4 py-2 text-white hover:bg-blue-600 disabled:opacity-50"
          onclick={onSubmit}
          disabled={submitting}
        >
          {submitting ? "Saving..." : submitLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
