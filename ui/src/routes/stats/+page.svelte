<script lang="ts">
  import { globalStore } from "$lib/stores/global";
  import { pb } from "$lib/pocketbase";

  interface Stats {
    qualifying_po_count: number;
    approved_expense_count: number;
    distinct_user_count: number;
  }

  let stats = $state<Stats | null>(null);
  let loading = $state(true);
  let error = $state("");

  const isAdmin = $derived($globalStore.claims.includes("admin"));

  async function fetchStats() {
    loading = true;
    error = "";
    try {
      const response = await pb.send("/api/stats", { method: "GET" });
      stats = response as Stats;
    } catch (err: any) {
      console.error("Failed to fetch stats:", err);
      error = err?.response?.message || "Failed to load stats";
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    if (isAdmin) {
      fetchStats();
    }
  });
</script>

{#if !isAdmin}
  <div class="p-4">
    <p class="text-red-600">Access denied. You must be an admin to view this page.</p>
  </div>
{:else}
  <div class="flex flex-col gap-4 p-4">
    <h1 class="text-2xl font-bold">Stats</h1>

    {#if loading}
      <div class="text-neutral-500">Loading...</div>
    {:else if error}
      <div class="text-red-600">{error}</div>
    {:else if stats}
      <h2 class="text-lg font-semibold">Deploy Readiness</h2>
      <p class="text-sm text-neutral-500">
        Approved expenses against new POs that have gone through the full approval flow in Turbo.
      </p>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div class="rounded-sm border border-neutral-300 bg-neutral-50 p-4">
          <div class="text-sm font-medium text-neutral-500">Qualifying POs</div>
          <div class="mt-1 text-3xl font-bold">{stats.qualifying_po_count}</div>
          <div class="mt-1 text-xs text-neutral-400">
            New POs that completed the approval flow
          </div>
        </div>
        <div class="rounded-sm border border-neutral-300 bg-neutral-50 p-4">
          <div class="text-sm font-medium text-neutral-500">Approved Expenses</div>
          <div class="mt-1 text-3xl font-bold">{stats.approved_expense_count}</div>
          <div class="mt-1 text-xs text-neutral-400">
            Against qualifying POs
          </div>
        </div>
        <div class="rounded-sm border border-neutral-300 bg-neutral-50 p-4">
          <div class="text-sm font-medium text-neutral-500">Distinct Users</div>
          <div class="mt-1 text-3xl font-bold">{stats.distinct_user_count}</div>
          <div class="mt-1 text-xs text-neutral-400">
            Who have touched these POs and expenses
          </div>
        </div>
      </div>
    {/if}
  </div>
{/if}
