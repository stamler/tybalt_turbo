<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import type { JobApiResponse } from "$lib/stores/jobs";
  import { pb } from "$lib/pocketbase";
  import { onMount } from "svelte";

  type LatestJob = JobApiResponse & { group_name: string };

  let items = $state<LatestJob[] | undefined>(undefined);
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);

  // Toggle: "projects" or "proposals" - default to proposals for latest jobs
  let jobType = $state<"projects" | "proposals">("proposals");

  const jobTypeOptions = [
    { id: "projects", label: "Projects" },
    { id: "proposals", label: "Proposals" },
  ];

  // Filter items based on job type
  const filteredItems = $derived.by(() => {
    if (!items) return undefined;
    return items.filter((job) => {
      const isProposal = job.number?.startsWith("P");
      return jobType === "proposals" ? isProposal : !isProposal;
    });
  });

  onMount(async () => {
    try {
      const res: LatestJob[] = await pb.send("/api/jobs/latest", { method: "GET" });
      items = res;
    } catch (e: any) {
      errorMessage = e?.response?.message ?? "Failed to load latest jobs";
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <div class="p-4">Loading…</div>
{:else if errorMessage}
  <div class="p-4 text-red-600">{errorMessage}</div>
{:else if filteredItems}
  <DsList items={filteredItems} search={true} inListHeader={jobType === "projects" ? "Latest Projects" : "Latest Proposals"}>
    {#snippet searchBarExtra()}
      <DSToggle bind:value={jobType} options={jobTypeOptions} />
    {/snippet}
    {#snippet anchor({ id, number }: LatestJob)}
      <a href="/jobs/{id}/details" class="font-bold hover:underline">{number}</a>
    {/snippet}
    {#snippet headline({ description }: LatestJob)}{description}{/snippet}
    {#snippet byline({ client }: LatestJob)}{client}{/snippet}
    {#snippet line1({ branch, manager }: LatestJob)}{#if branch}<DsLabel color="neutral">{branch}</DsLabel>{/if}{#if manager}<DsLabel color="purple">{manager}</DsLabel>{/if}{/snippet}
    {#snippet actions({ id }: LatestJob)}
      <DsActionButton action="/jobs/{id}/edit" icon="mdi:pencil" title="Edit" color="blue" />
    {/snippet}
  </DsList>
{/if}
