<script lang="ts">
  import { globalStore } from "$lib/stores/global";
  import { jobRequestsEnabled } from "$lib/stores/appConfig";
  import { wholeCAD, canViewJobCreateQueue } from "$lib/jobRequests";
  let { data } = $props();
  const titles: Record<string, string> = {
    mine: "Project Requests",
    all: "All Project Requests",
    approval: "Request Approvals",
    creation: "Job Create Queue",
    abandoned: "Abandoned Requests",
  };
</script>

<div class="mx-auto max-w-6xl space-y-5 p-4">
  <div class="flex items-center justify-between gap-4">
    <h1 class="text-2xl font-bold">{titles[data.queue] || "Project Requests"}</h1>
    {#if $jobRequestsEnabled && $globalStore.claims.includes("time")}<a
        class="rounded bg-green-700 px-4 py-2 text-white"
        href="/job-requests/new">New Project Request</a
      >{/if}
  </div>
  <nav class="flex flex-wrap gap-4 text-sm" aria-label="Request lists">
    <a class="text-blue-700 underline" href="?queue=mine">My requests</a
    >{#if $globalStore.claims.includes("job")}<a class="text-blue-700 underline" href="?queue=all"
        >All requests</a
      >{/if}{#if $jobRequestsEnabled && $globalStore.claims.includes("project_authorization")}<a
        class="text-blue-700 underline"
        href="?queue=approval">Request approvals</a
      >{/if}{#if canViewJobCreateQueue($globalStore.claims, $jobRequestsEnabled, $globalStore.attentionCounts.counts)}<a
        class="text-blue-700 underline"
        href="?queue=creation">Job Create Queue</a
      >{/if}{#if $globalStore.claims.includes("admin")}<a
        class="text-blue-700 underline"
        href="?queue=abandoned">Abandoned requests</a
      >{/if}
  </nav>
  {#if !$jobRequestsEnabled}<p class="rounded border border-amber-300 bg-amber-50 p-3">
      Request processing is disabled. Saved requests remain available.
    </p>{/if}
  {#if data.queue === "abandoned"}<p>
      These requests have no saved edits in the last 60 days. Review them with the creator before
      you delete them.
    </p>{/if}
  {#if data.queue === "creation"}<p>
      Review each approved request and its PDF. Check existing jobs and pending requests before you
      create the job.
    </p>{/if}
  <p class="text-sm text-neutral-600">{data.totalItems ?? data.items?.length ?? 0} requests</p>
  <div class="overflow-x-auto rounded border border-neutral-300">
    <table class="w-full text-left">
      <thead class="bg-neutral-100"
        ><tr
          ><th>Request reference</th><th>Project / scope of work</th><th>State</th><th
            >Project value</th
          ></tr
        ></thead
      ><tbody>
        {#each data.items ?? [] as item}<tr class="border-t border-neutral-200"
            ><td
              ><a class="text-blue-700 underline" href={`/job-requests/${item.id}`}>{item.id}</a
              ></td
            ><td
              >{item.description || "Untitled request"}{#if item.parent}<span
                  class="ml-2 text-sm text-neutral-500">Subjob</span
                >{/if}</td
            ><td>{item.status || item.state}</td><td>{wholeCAD(item.project_value)}</td></tr
          >{:else}<tr><td colspan="4">No requests in this list.</td></tr>{/each}
      </tbody>
    </table>
  </div>
  <nav class="flex gap-4" aria-label="Request pages">
    {#if data.page > 1}<a
        href={`?queue=${data.queue}&page=${data.page - 1}`}
        class="text-blue-700 underline">Previous</a
      >{/if}{#if data.page < data.totalPages}<a
        href={`?queue=${data.queue}&page=${data.page + 1}`}
        class="text-blue-700 underline">Next</a
      >{/if}
  </nav>
</div>

<style>
  th,
  td {
    padding: 0.75rem;
  }
</style>
