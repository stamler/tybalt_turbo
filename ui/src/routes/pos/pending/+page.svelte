<script lang="ts">
  import type { PageData } from "./$types";
  import PurchaseOrdersList from "$lib/components/PurchaseOrdersList.svelte";
  let { data }: { data: PageData } = $props();
</script>

<PurchaseOrdersList inListHeader="Purchase Orders Pending My Approval" {data} />
