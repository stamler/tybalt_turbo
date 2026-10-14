<script lang="ts">
  import { JobValue } from "$lib/jobValue.svelte";
  import JobValueLine from "$lib/components/jobs/JobValueLine.svelte";
  import JobValueProposal from "$lib/components/jobs/JobValueProposal.svelte";
  import JobValueQueue from "$lib/components/jobs/JobValueQueue.svelte";
  import JobValueStatus from "$lib/components/jobs/JobValueStatus.svelte";
  import { formatCurrency } from "$lib/utilities";
  // Mounts the value components as the job page arranges them, or the queue.
  let {
    jobId,
    effectiveValue = 0,
    queue = false,
  }: { jobId?: string; effectiveValue?: number; queue?: boolean } = $props();
  // svelte-ignore state_referenced_locally
  const value = new JobValue(jobId);
  $effect(() => {
    if (!queue) value.reload();
  });
</script>

{#if queue}<JobValueQueue />{:else}
  <JobValueStatus {value} />
  <div class="flex items-center gap-2">
    <span class="font-semibold">Project Value:</span>
    {formatCurrency(effectiveValue)}
    <JobValueLine {value} />
  </div>
  <JobValueProposal {value} {effectiveValue} />
{/if}
