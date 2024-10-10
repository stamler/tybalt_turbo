<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import { globalStore } from "$lib/stores/global";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
</script>

<DsList items={$globalStore.jobs} search={true}>
  {#snippet anchor({ number })}{number}{/snippet}
  {#snippet headline({ description })}{description}{/snippet}
  {#snippet byline({ expand })}
    <span>{expand?.client.name}</span>
  {/snippet}
  {#snippet line1({ expand })}
    <span class="flex gap-1">
      {#each expand?.categories_via_job as category}
        <span class="rounded-sm border border-cyan-100 bg-cyan-50 px-1">{category.name}</span>
      {/each}
    </span>
  {/snippet}
  {#snippet actions({ id })}
    <DsActionButton action="/jobs/{id}/edit" icon="mdi:pencil" title="Edit" color="blue" />
    <DsActionButton
      action="/details/{id}"
      icon="mdi:more-circle"
      title="More Details"
      color="blue"
    />
    <DsActionButton
      action={() => globalStore.deleteItem("jobs", id)}
      icon="mdi:delete"
      title="Delete"
      color="red"
    />
  {/snippet}
</DsList>
