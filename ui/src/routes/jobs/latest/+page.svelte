<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import type { JobApiResponse } from "$lib/stores/jobs";
  import { pb } from "$lib/pocketbase";
  import { onMount } from "svelte";

  type LatestJob = JobApiResponse & { group_name: string };

  let items = $state<LatestJob[] | undefined>(undefined);
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);

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
{:else if items}
  <DsList {items} search={true} inListHeader="Latest Jobs" groupField="group_name">
    {#snippet anchor({ id, number }: LatestJob)}
      <a href="/jobs/{id}/details" class="font-bold hover:underline">{number}</a>
    {/snippet}
    {#snippet headline({ description }: LatestJob)}{description}{/snippet}
    {#snippet byline({ client }: LatestJob)}{client}{/snippet}
    {#snippet line1({ branch }: LatestJob)}{#if branch}<DsLabel color="neutral">{branch}</DsLabel
        >{/if}{/snippet}
    {#snippet actions({ id }: LatestJob)}
      <DsActionButton action="/jobs/{id}/edit" icon="mdi:pencil" title="Edit" color="blue" />
    {/snippet}
  </DsList>
{/if}
