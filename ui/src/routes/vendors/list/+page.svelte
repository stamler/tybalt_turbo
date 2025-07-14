<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { VendorApiResponse } from "$lib/stores/vendors";
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
    {#snippet headline({ id, name, alias }: VendorApiResponse)}
      <span class="flex items-center gap-2">
        <a href={`/vendors/${id}/details`} class="text-blue-600 hover:underline">
          {name}
        </a>
        {#if alias !== ""}
          <span class="opacity-30">({alias})</span>
        {/if}
      </span>
    {/snippet}

    {#snippet byline({ expenses_count, purchase_orders_count }: VendorApiResponse)}
      <span class="mr-2">
        {expenses_count === 0
          ? "no expenses"
          : expenses_count === 1
            ? "1 expense"
            : `${expenses_count} expenses`}
        •
        {purchase_orders_count === 0
          ? "no POs"
          : purchase_orders_count === 1
            ? "1 PO"
            : `${purchase_orders_count} POs`}
      </span>
    {/snippet}

    {#snippet actions({ id }: VendorApiResponse)}
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
