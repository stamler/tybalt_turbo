<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { VendorsResponse } from "$lib/pocketbase-types";
  import { pb } from "$lib/pocketbase";
  import { vendors } from "$lib/stores/vendors";

  // initialize the stores, noop if already initialized
  vendors.init();
</script>

{#if $vendors.index !== null}
  <DsSearchList
    index={$vendors.index}
    inListHeader="Vendors"
    fieldName="vendor"
    uiName="search vendors..."
    collectionName="vendors"
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
        action={`/vendors/${id}/absorb`}
        icon="mdi:merge"
        title="Absorb other vendors into this one"
        color="yellow"
      />
      <DsActionButton
        action={() => pb.collection("vendors").delete(id)}
        icon="mdi:delete"
        title="Delete"
        color="red"
      />
    {/snippet}
  </DsSearchList>
{/if}
