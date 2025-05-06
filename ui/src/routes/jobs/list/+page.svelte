<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import { globalStore } from "$lib/stores/global";
  import type { JobsResponse } from "$lib/pocketbase-types";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
</script>

{#if $globalStore.jobsIndex !== null}
  <DsSearchList index={$globalStore.jobsIndex} fieldName="job" uiName="search...">
    {#snippet anchor({ number }: JobsResponse)}{number}{/snippet}
    {#snippet headline({ description }: JobsResponse)}{description}{/snippet}
    {#snippet byline({ expand }: JobsResponse)}{expand?.client.name}{/snippet}
    {#snippet line1({ expand }: JobsResponse)}
      {#each expand?.categories_via_job as category}
        <span class="rounded-sm border border-cyan-100 bg-cyan-50 px-1">{category.name}</span>
      {/each}
    {/snippet}
    {#snippet actions({ id }: JobsResponse)}
      <DsActionButton action="/jobs/{id}/edit" icon="mdi:pencil" title="Edit" color="blue" />
      <DsActionButton
        action="/details/{id}"
        icon="mdi:more-circle"
        title="More Details"
        color="blue"
      />
    {/snippet}
  </DsSearchList>
{/if}
