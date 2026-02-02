<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { RateSheetResponse } from "$lib/stores/rateSheets";
  import { rateSheets } from "$lib/stores/rateSheets";
  import { shortDate } from "$lib/utilities";

  // initialize the store, noop if already initialized
  rateSheets.init();
</script>

{#if $rateSheets.items.length > 0 || $rateSheets.initialized}
  <DsList items={$rateSheets.items} search={true} inListHeader="Rate Sheets">
    {#snippet headline({ id, name }: RateSheetResponse)}
      <a href={`/rate-sheets/${id}/details`} class="text-blue-600 hover:underline">
        {name}
      </a>
    {/snippet}

    {#snippet byline({ effective_date, revision, job_count }: RateSheetResponse)}
      <span class="text-neutral-500">
        rev. {revision} • {shortDate(effective_date, true)} • {job_count} job{job_count === 1
          ? ""
          : "s"}
      </span>
    {/snippet}

    {#snippet line1({ active }: RateSheetResponse)}
      {#if active}
        <span
          class="inline-flex items-center rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800"
        >
          Active
        </span>
      {:else}
        <span
          class="inline-flex items-center rounded-full bg-neutral-100 px-2 py-0.5 text-xs font-medium text-neutral-600"
        >
          Inactive
        </span>
      {/if}
    {/snippet}

    {#snippet actions({ id, active }: RateSheetResponse)}
      {#if active}
        <DsActionButton
          action={`/rate-sheets/copy?revise=${id}`}
          icon="mdi:file-replace-outline"
          title="Revise"
          color="blue"
        />
      {/if}
      <DsActionButton
        action={`/rate-sheets/copy?from=${id}`}
        icon="mdi:content-copy"
        title="Use as Template"
        color="green"
      />
    {/snippet}
  </DsList>
{:else if $rateSheets.loading}
  <div class="p-4 text-neutral-500">Loading rate sheets...</div>
{/if}
