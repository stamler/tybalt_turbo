<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { globalStore } from "$lib/stores/global";
  import type { VendorsResponse } from "$lib/pocketbase-types";
</script>

{#if $globalStore.vendorsIndex !== null}
  <DsSearchList
    index={$globalStore.vendorsIndex}
    inListHeader="Vendors"
    fieldName="vendor"
    uiName="search vendors..."
  >
    {#snippet headline({ name, alias }: VendorsResponse)}
      <span class="flex items-center gap-2">
        {name}
        {#if alias !== ""}
          <span class="opacity-30">({alias})</span>
        {/if}
      </span>
    {/snippet}

    {#snippet actions({ id }: VendorsResponse)}
      <DsActionButton
        action={`/vendors/${id}/edit`}
        icon="mdi:edit-outline"
        title="Edit"
        color="blue"
      />
      <DsActionButton
        action={() => globalStore.deleteItem("vendors", id)}
        icon="mdi:delete"
        title="Delete"
        color="red"
      />
    {/snippet}
  </DsSearchList>
{/if}
