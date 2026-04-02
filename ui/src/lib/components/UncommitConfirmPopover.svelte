<script lang="ts">
  import DSPopover from "./DSPopover.svelte";

  let {
    show = $bindable(),
    recordLabel,
    submitting = false,
    error = null,
    onSubmit,
    onCancel,
  }: {
    show: boolean;
    recordLabel: string;
    submitting?: boolean;
    error?: string | null;
    onSubmit: () => void;
    onCancel: () => void;
  } = $props();
</script>

<DSPopover
  bind:show
  title="Confirm Uncommit"
  subtitle={`This will return the ${recordLabel} to an approved, uncommitted state.`}
  {error}
  {submitting}
  submitLabel="Confirm Uncommit"
  {onSubmit}
  {onCancel}
>
  <div class="rounded-sm border border-red-300 bg-red-50 p-3 text-red-900">
    <p class="font-semibold">
      Please confirm with accounting and any other report consumers before continuing.
    </p>
    <p class="mt-2 text-sm">
      Uncommitting this {recordLabel} will change downstream reports, so make sure anyone relying on
      those numbers is aligned first.
    </p>
    <p class="mt-2 text-sm">
      If this {recordLabel} has already written back to a legacy system, that writeback must be
      cleaned up manually after the uncommit.
    </p>
  </div>

  <div class="rounded-sm border border-amber-300 bg-amber-50 p-3 text-amber-900">
    <p class="font-semibold">Review what will happen if you continue:</p>
    <ul class="mt-2 list-disc space-y-1 pl-5 text-sm">
      <li>The current committed state will be cleared.</li>
      <li>The {recordLabel} will return to an approved, uncommitted state.</li>
      <li>No uncommit request is sent until you confirm.</li>
    </ul>
  </div>
</DSPopover>
