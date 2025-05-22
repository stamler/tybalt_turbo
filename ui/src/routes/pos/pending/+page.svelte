<script lang="ts">
  import type { PurchaseOrdersPageData } from "$lib/svelte-types";
  import PurchaseOrdersList from "$lib/components/PurchaseOrdersList.svelte";
  let { data }: { data: PurchaseOrdersPageData } = $props();
</script>

<PurchaseOrdersList inListHeader="Purchase Orders Pending My Approval" {data} />
