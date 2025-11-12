<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { JobApiResponse } from "$lib/stores/jobs";
  import { pb } from "$lib/pocketbase";
  import { onMount } from "svelte";
  import { JobsStatusOptions } from "$lib/pocketbase-types";

  type StaleJob = JobApiResponse & { last_reference: string; last_reference_type: string };

  let items = $state<StaleJob[] | undefined>(undefined);
  let loading = $state(false);
  let errorMessage = $state<string | null>(null);
  let prefix = $state("");
  let age = $state(180);
  let mutatingById = $state<Record<string, boolean>>({});

  const load = async () => {
    const p = prefix.trim();
    loading = true;
    errorMessage = null;
    try {
      const qp = new URLSearchParams();
      if (p) qp.set("prefix", p);
      if (age && age > 0) qp.set("age", String(age));
      const res: StaleJob[] = await pb.send(`/api/jobs/stale?${qp.toString()}`, {
        method: "GET",
      });
      items = res;
    } catch (e: any) {
      errorMessage = e?.response?.message ?? "Failed to load stale jobs";
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
    // intentionally not auto-loading (mirror Unused page behavior)
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
  <input
    type="number"
    min="1"
    bind:value={age}
    class="w-24 rounded border border-neutral-300 px-2 py-1 text-base"
    title="Age in days"
    onkeydown={(e) => e.key === "Enter" && load()}
  />
  <DsActionButton action={load} icon="mdi:magnify" title="Load" color="yellow" />
</div>

{#if loading}
  <div class="p-4">Loading…</div>
{:else if errorMessage}
  <div class="p-4 text-red-600">{errorMessage}</div>
{:else if items}
  <DsList {items} search={true} inListHeader="Stale Jobs">
    {#snippet anchor({ id, number }: StaleJob)}
      <a href="/jobs/{id}/details" class="font-bold hover:underline">{number}</a>
    {/snippet}
    {#snippet headline({ description }: StaleJob)}{description}{/snippet}
    {#snippet byline({ client }: StaleJob)}{client}{/snippet}
    {#snippet line1({ last_reference, last_reference_type }: StaleJob)}
      <span class="text-sm text-neutral-600">
        Last activity: {last_reference}
        {#if last_reference_type}
          — {last_reference_type.replace("_", " ")}
        {/if}
      </span>
    {/snippet}
    {#snippet actions({ id, number }: StaleJob)}
      <span class="mr-1 text-sm text-neutral-600">Set Status:</span>
      {#if number?.startsWith("P")}
        <DsActionButton
          action={() => setStatus(id, JobsStatusOptions["Not Awarded"])}
          icon="mdi:trophy-broken"
          title="Not-Awarded"
          color="gray"
          loading={!!mutatingById[id]}
        />
        <DsActionButton
          action={() => setStatus(id, JobsStatusOptions.Cancelled)}
          icon="mdi:cancel"
          title="Cancelled"
          color="red"
          loading={!!mutatingById[id]}
        />
      {:else}
        <DsActionButton
          action={() => setStatus(id, JobsStatusOptions.Closed)}
          icon="mdi:door"
          title="Set Closed"
          color="blue"
          loading={!!mutatingById[id]}
        />
        <DsActionButton
          action={() => setStatus(id, JobsStatusOptions.Cancelled)}
          icon="mdi:cancel"
          title="Set Cancelled"
          color="red"
          loading={!!mutatingById[id]}
        />
      {/if}
    {/snippet}
  </DsList>
{/if}
