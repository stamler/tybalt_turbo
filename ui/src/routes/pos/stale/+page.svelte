<script lang="ts">
  import type { PurchaseOrdersPageData } from "$lib/svelte-types";
  import PurchaseOrdersList from "$lib/components/PurchaseOrdersList.svelte";
  let { data }: { data: PurchaseOrdersPageData } = $props();
  const staleDate = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split("T")[0];
</script>

<PurchaseOrdersList inListHeader={`Stale Purchase Orders (Active before ${staleDate})`} {data} />
