<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import WIPSummary from "./WIPSummary.svelte";
  import type { JobWIP } from "./wip";
  let { jobId }: { jobId: string } = $props();
  const requestId = $props.id();
  let data = $state<JobWIP | null>(null);
  let error = $state(false);
  let retry = $state(0);
  $effect(() => {
    const id = jobId;
    void retry;
    data = null;
    error = false;
    let current = true;
    const controller = new AbortController();
    pb.send<JobWIP>(`/api/jobs/${id}/wip`, {
      method: "GET",
      signal: controller.signal,
      requestKey: requestId,
    })
      .then((result) => {
        if (current) data = result;
      })
      .catch(() => {
        if (current) error = true;
      });
    return () => {
      current = false;
      controller.abort();
    };
  });
</script>

<div class="px-4 py-3">
  {#if error}
    <div role="alert" class="flex items-center gap-3 text-red-700">
      <p>Failed to load WIP.</p>
      <button type="button" class="rounded-sm border px-3 py-1" onclick={() => retry++}
        >Try again</button
      >
    </div>
  {:else if data}
    <WIPSummary {data} />
  {:else}
    <p role="status" class="text-neutral-600">Loading WIP…</p>
  {/if}
</div>
