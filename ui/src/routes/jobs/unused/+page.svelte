<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import type { JobApiResponse } from "$lib/stores/jobs";
  import { pb } from "$lib/pocketbase";
  import { onMount } from "svelte";
  import { JobsStatusOptions } from "$lib/pocketbase-types";

  let items = $state<JobApiResponse[] | undefined>(undefined);
  let loading = $state(false);
  let errorMessage = $state<string | null>(null);
  let prefix = $state("");
  let mutatingById = $state<Record<string, boolean>>({});

  const load = async () => {
    const p = prefix.trim();
    if (!p) {
      errorMessage = "Please enter a prefix (e.g., 25- or P23-).";
      items = undefined;
      return;
    }
    loading = true;
    errorMessage = null;
    try {
      const res: JobApiResponse[] = await pb.send(
        `/api/jobs/unused?prefix=${encodeURIComponent(p)}`,
        {
          method: "GET",
        },
      );
      items = res;
    } catch (e: any) {
      errorMessage = e?.response?.message ?? "Failed to load unused active jobs";
      items = undefined;
    } finally {
      loading = false;
    }
  };

  const setStatus = async (jobId: string, status: JobsStatusOptions) => {
    mutatingById = { ...mutatingById, [jobId]: true };
    try {
      await pb.collection("jobs").update(jobId, { status });
      await load();
    } catch (e) {
      console.error("Failed to update job status", e);
    } finally {
      mutatingById = { ...mutatingById, [jobId]: false };
    }
  };

  onMount(() => {
    // no auto-load; wait for user to enter prefix
  });
</script>

<div class="flex items-center gap-x-2 bg-neutral-200 p-2">
  <input
    type="text"
    placeholder="Enter job number prefix (e.g., 25- or P23-)"
    bind:value={prefix}
    class="flex-1 rounded border border-neutral-300 px-1 py-1 text-base max-[639px]:px-2 max-[639px]:py-2 max-[639px]:text-lg"
    onkeydown={(e) => e.key === "Enter" && load()}
  />
  <DsActionButton action={load} icon="mdi:magnify" title="Load" color="yellow" />
</div>

{#if loading}
  <div>Loading…</div>
{:else if errorMessage}
  <div class="text-red-600">{errorMessage}</div>
{:else if items}
  <DsList {items} search={true} inListHeader="Unused Active Jobs">
    {#snippet anchor({ id, number }: JobApiResponse)}
      <a href="/jobs/{id}/details" class="font-bold hover:underline">{number}</a>
    {/snippet}
    {#snippet headline({ description }: JobApiResponse)}{description}{/snippet}
    {#snippet byline({ client }: JobApiResponse)}{client}{/snippet}
    {#snippet line1({ branch }: JobApiResponse)}{#if branch}<DsLabel color="neutral">{branch}</DsLabel>{/if}{/snippet}
    {#snippet actions({ id, number }: JobApiResponse)}
      {#if number?.startsWith("P")}
        <DsActionButton
          action={() => setStatus(id, JobsStatusOptions.Awarded)}
          icon="mdi:trophy"
          title="Awarded"
          color="yellow"
          loading={!!mutatingById[id]}
        />
        <DsActionButton
          action={() => setStatus(id, JobsStatusOptions["Not Awarded"])}
          icon="mdi:trophy-broken"
          title="Not Awarded"
          color="gray"
          loading={!!mutatingById[id]}
        />
      {:else}
        <DsActionButton
          action={() => setStatus(id, JobsStatusOptions.Closed)}
          icon="mdi:door"
          title="Closed"
          color="blue"
          loading={!!mutatingById[id]}
        />
      {/if}
      <DsActionButton
        action={() => setStatus(id, JobsStatusOptions.Cancelled)}
        icon="mdi:cancel"
        title="Cancel"
        color="red"
        loading={!!mutatingById[id]}
      />
    {/snippet}
  </DsList>
{/if}
