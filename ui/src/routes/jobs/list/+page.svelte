<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import { jobs } from "$lib/stores/jobs";
  // TODO: JobsResponse isn't actually the correct type for the items in the
  // index, but hobbles along for now
  import type { JobApiResponse } from "$lib/stores/jobs";
  import DsActionButton from "$lib/components/DSActionButton.svelte";

  // initialize the jobs store, noop if already initialized
  jobs.init();
</script>

{#if $jobs.index !== null}
  <DsSearchList index={$jobs.index} inListHeader="Jobs" fieldName="job" uiName="search jobs...">
    {#snippet anchor({ number }: JobApiResponse)}{number}{/snippet}
    {#snippet headline({ description }: JobApiResponse)}{description}{/snippet}
    {#snippet byline({ client }: JobApiResponse)}{client}{/snippet}
    {#snippet actions({ id }: JobApiResponse)}
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
