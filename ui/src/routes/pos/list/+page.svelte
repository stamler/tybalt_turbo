<script lang="ts">
  import type { PageData } from "./$types";
  import PurchaseOrdersList from "$lib/components/PurchaseOrdersList.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import type { VisiblePurchaseOrderResponse } from "$lib/poVisibility";

  let { data }: { data: PageData } = $props();

  type PurchaseOrderListMode = "active_unapproved" | "closed_cancelled";

  let purchaseOrderListMode = $state<PurchaseOrderListMode>("active_unapproved");

  const listModeOptions = [
    { id: "active_unapproved", label: "Active/Unapproved" },
    { id: "closed_cancelled", label: "Closed/Cancelled" },
  ];

  const statusFilter = $derived((po: VisiblePurchaseOrderResponse) => {
    if (purchaseOrderListMode === "closed_cancelled") {
      return po.status === "Closed" || po.status === "Cancelled";
    }
    return po.status === "Active" || po.status === "Unapproved";
  });
</script>

<PurchaseOrdersList
  inListHeader={purchaseOrderListMode === "closed_cancelled"
    ? "My Purchase Orders - Closed/Cancelled"
    : "My Purchase Orders - Active/Unapproved"}
  {data}
  showRemaining={true}
  filter={statusFilter}
>
  {#snippet searchBarExtra()}
    <div class="flex items-center gap-2 max-[639px]:w-full max-[639px]:flex-wrap">
      <DSToggle
        bind:value={purchaseOrderListMode}
        options={listModeOptions}
        ariaLabel="Purchase order status filter"
      />
    </div>
  {/snippet}
</PurchaseOrdersList>
