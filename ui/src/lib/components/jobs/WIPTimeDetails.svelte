<script lang="ts">
  import { resolve } from "$app/paths";
  import { pb } from "$lib/pocketbase";
  import type { JobWIP } from "./wip";
  let { data, jobId = "" }: { data: JobWIP; jobId?: string } = $props();
  type MissingRates = {
    rate_sheet: { id: string; name: string; revision: number } | null;
    roles: { id: string; name: string; hours: number }[];
  };
  const id = $props.id();
  let request = $state.raw<{ data: JobWIP; jobId: string } | null>(null);
  let details = $state<MissingRates | null>(null);
  let loading = $state(false);
  let error = $state(false);
  const load = () => {
    request = { data, jobId };
  };

  $effect(() => {
    details = null;
    error = false;
    loading = false;
    if (!request || request.data !== data || request.jobId !== jobId || !jobId) return;
    const controller = new AbortController();
    let current = true;
    loading = true;
    pb.send<MissingRates>(`/api/jobs/${jobId}/wip/missing-rates`, {
      method: "GET",
      query: { as_of: data.as_of },
      requestKey: id,
      signal: controller.signal,
    })
      .then((result) => {
        if (current) details = result;
      })
      .catch(() => {
        if (current) error = true;
      })
      .finally(() => {
        if (current) loading = false;
      });
    return () => {
      current = false;
      controller.abort();
    };
  });
</script>

<p><strong>{data.hours.toFixed(2)}</strong> recorded hours.</p>
{#if data.no_rate_sheet}
  <p>This job has no rate sheet. Time uses employee default charge-out rates where available.</p>
{:else}
  {#if data.missing_role_hours > 0}
    <p>
      <strong>{data.missing_role_hours.toFixed(2)}</strong> hours have no role recorded on the time entry.
    </p>
  {/if}
  {#if data.missing_role_rate_hours > 0}
    <p>
      <strong>{data.missing_role_rate_hours.toFixed(2)}</strong> hours have a role with no matching
      rate in the
      {#if details?.rate_sheet}
        <a
          class="text-blue-700 underline"
          href={resolve("/rate-sheets/[id]/details", { id: details.rate_sheet.id })}
          >{details.rate_sheet.name}</a
        >
        (revision {details.rate_sheet.revision})
      {:else}
        assigned
      {/if}
      rate sheet.
      {#if jobId && !details}
        <button
          type="button"
          class="text-blue-700 underline disabled:text-neutral-500"
          onclick={load}
          disabled={loading || details !== null}
          aria-expanded={details !== null}
          aria-controls={`${id}-rates`}>missing rates</button
        >
      {/if}
    </p>
    <div id={`${id}-rates`} aria-live="polite" class="space-y-2">
      {#if loading}<p role="status">Loading missing rates…</p>
      {:else if error}
        <p role="alert">
          Could not load missing rates. <button
            type="button"
            class="text-blue-700 underline"
            onclick={load}>Try again</button
          >
        </p>
      {:else if details}
        {#if details.rate_sheet}
          {#if details.roles.length}
            <ul class="list-disc space-y-1 pl-5">
              {#each details.roles as role (role.id)}
                <li>{role.name} — {role.hours.toFixed(2)} hours</li>
              {/each}
            </ul>
          {:else}<p>No missing role rates found. The job or rate sheet may have changed.</p>{/if}
        {:else}<p>This job no longer has an assigned rate sheet.</p>{/if}
      {/if}
    </div>
  {/if}
{/if}
{#if data.estimated_hours > 0}
  <p>
    <strong>{data.estimated_hours.toFixed(2)}</strong> hours use employee default charge-out rates. Their
    value is estimated.
  </p>
{/if}
{#if data.unpriced_hours > 0}
  <p>
    <strong>{data.unpriced_hours.toFixed(2)}</strong> hours cannot be priced because no sheet rate or
    positive employee default rate is available. Their value is excluded.
  </p>
{/if}
