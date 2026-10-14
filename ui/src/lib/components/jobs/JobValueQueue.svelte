<script lang="ts">
  import { resolve } from "$app/paths";
  import { JobValue } from "$lib/jobValue.svelte";
  import JobValueReview from "./JobValueReview.svelte";
  // The signed-in approver's pending value changes.
  const value = new JobValue(undefined, true);
  $effect(() => {
    value.reload();
  });
</script>

<section class="space-y-4">
  {#if value.error}<p role="alert" class="rounded bg-red-50 p-3 text-red-800">
      {value.error}
    </p>{/if}
  {#if value.loading && !value.items.length}<p>Loading value changes…</p>{/if}
  {#each value.items as item (item.id)}
    <article class="space-y-3 rounded-sm border border-neutral-300 p-4">
      <div>
        <a href={resolve(`/jobs/${item.job}/details`)} class="font-semibold text-blue-700 underline"
          >{item.job_number || item.job} — {item.job_description || "Job details"}</a
        >
        <p class="text-sm text-neutral-600">
          Client: {item.related?.client?.name || "Not provided"} · Manager: {item.related?.manager
            ?.name || "Not provided"}
        </p>
      </div>
      <JobValueReview {value} {item} />
    </article>
  {:else}
    {#if !value.loading}<p>No value changes are waiting for your approval.</p>{/if}
  {/each}
</section>
