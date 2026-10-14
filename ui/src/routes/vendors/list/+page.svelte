<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { VendorApiResponse } from "$lib/stores/vendors";
  import { pb } from "$lib/pocketbase";
  import { vendors } from "$lib/stores/vendors";
  import { expensesEditingEnabled } from "$lib/stores/appConfig";
  import { globalStore } from "$lib/stores/global";
  import { trimmedOrEmpty } from "$lib/utilities";

  // initialize the stores, noop if already initialized
  vendors.init();

  async function del(id: string, name: string): Promise<void> {
    try {
      await pb.collection("vendors").delete(id);
    } catch (error: unknown) {
      // PocketBase answers 404 when the delete rule refuses the request: the
      // user is not a payables admin, or a purchase order or expense started
      // using the vendor after the list loaded.
      const status = (error as { status?: number })?.status;
      globalStore.addError(
        status === 404
          ? `${name} could not be deleted. Only payables admins can delete vendors, and a vendor that purchase orders or expenses use must be absorbed into another vendor instead.`
          : `${name} could not be deleted: ${(error as { message?: string })?.message ?? "unknown error"}`,
      );
    }
  }
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
        {#if trimmedOrEmpty(alias)}
          <span class="opacity-30">({trimmedOrEmpty(alias)})</span>
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

    {#snippet actions({ id, name, referenced }: VendorApiResponse)}
      {#if $expensesEditingEnabled}
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
        <!-- Vendors that purchase orders or expenses use cannot be deleted; absorb them instead. -->
        {#if !referenced}
          <DsActionButton
            action={() => del(id, name)}
            icon="mdi:delete"
            title="Delete"
            color="red"
          />
        {/if}
      {/if}
    {/snippet}
  </DsSearchList>
{/if}
