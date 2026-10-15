<script lang="ts">
  import type { WorkflowRecord } from "$lib/clientEditors";
  import { JobValue, valueCAD, valueDate } from "$lib/jobValue.svelte";
  // The selected approver's review of one pending value change. Each instance
  // keeps its own confirmation and rejection reason.
  let { value, item, ondone }: { value: JobValue; item: WorkflowRecord; ondone?: () => void } =
    $props();
  // The revision under review. A correction, such as a replacement PDF, saves a
  // new version, which clears the confirmation and reason given for the earlier
  // one, so approval always follows a review of the data it sends.
  const revision = $derived(`${item.id}:${item.version}`);
  let reviewed = $derived.by(() => {
      void revision;
      return false;
    }),
    reason = $derived.by(() => {
      void revision;
      return "";
    });
  async function decide(name: "approve" | "reject") {
    if (await value.act(item, name, name === "reject" ? reason : "")) ondone?.();
  }
</script>

<div class="space-y-3">
  <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
    <dt class="font-semibold">Current value</dt>
    <dd>{valueCAD(item.effective_project_value)}</dd>
    <dt class="font-semibold">Proposed value</dt>
    <dd>{valueCAD(item.project_value)}</dd>
    <dt class="font-semibold">Proposed by</dt>
    <dd>{item.uploader_name || item.uploader} on {valueDate(item.uploaded)}</dd>
  </dl>
  <button type="button" class="text-sm text-blue-700 underline" onclick={() => value.pdf(item)}
    >Open authorization PDF</button
  >
  <label class="flex items-start gap-2 text-sm"
    ><input type="checkbox" class="mt-1" bind:checked={reviewed} />I reviewed this PDF and its full
    proposed value.</label
  >
  <button
    type="button"
    disabled={value.busy || !reviewed}
    class="rounded bg-green-700 px-3 py-2 text-white disabled:opacity-40"
    onclick={() => decide("approve")}>Approve value</button
  >
  <label class="block text-sm font-semibold"
    >Rejection reason<textarea
      class="mt-1 block w-full rounded border border-neutral-400 p-2 font-normal"
      bind:value={reason}></textarea></label
  >
  <button
    type="button"
    disabled={value.busy || !reason.trim()}
    class="rounded border border-red-700 px-3 py-2 text-red-800 disabled:opacity-40"
    onclick={() => decide("reject")}>Reject</button
  >
</div>
