<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import PurchaseOrdersList from "$lib/components/PurchaseOrdersList.svelte";
  import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";
  import { fetchVisiblePOs } from "$lib/poVisibility";
  import { onMount } from "svelte";

  interface PurchaseOrdersListData {
    items?: PurchaseOrdersAugmentedResponse[];
    realtime_source?: "visible" | "pending";
  }

  let days = $state(30);
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);
  let expiringBefore = $state("");
  let listKey = $state(0);
  let listData = $state<PurchaseOrdersListData>({
    items: undefined,
    realtime_source: "visible",
  });

  function computeExpiringBefore(daysAhead: number): string {
    const target = new Date();
    target.setHours(0, 0, 0, 0);
    target.setDate(target.getDate() + daysAhead);
    return target.toISOString().split("T")[0];
  }

  async function load(): Promise<void> {
    const safeDays = Number.isFinite(days) && days > 0 ? Math.floor(days) : 30;
    days = safeDays;
    expiringBefore = computeExpiringBefore(safeDays);

    loading = true;
    errorMessage = null;
    try {
      const result = await fetchVisiblePOs("expiring", expiringBefore);
      listData = {
        items: result,
        realtime_source: "visible",
      };
      listKey += 1;
    } catch (e: any) {
      errorMessage = e?.response?.message ?? "Failed to load expiring purchase orders";
      listData = {
        items: [],
        realtime_source: "visible",
      };
      listKey += 1;
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<div class="flex items-center gap-x-2 bg-neutral-200 p-2">
  <input
    type="number"
    min="1"
    bind:value={days}
    class="w-24 rounded-sm border border-neutral-300 px-2 py-1 text-base"
    title="Days ahead"
    onkeydown={(e) => e.key === "Enter" && load()}
  />
  <DsActionButton action={load} icon="mdi:magnify" title="Load" color="yellow" />
</div>

{#if loading}
  <div class="p-4">Loading…</div>
{:else if errorMessage}
  <div class="p-4 text-red-600">{errorMessage}</div>
{:else}
  {#key listKey}
    <PurchaseOrdersList
      inListHeader={`Expiring Recurring Purchase Orders (end date on or before ${expiringBefore})`}
      data={listData}
    />
  {/key}
{/if}
