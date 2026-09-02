<script lang="ts">
  import { resolve } from "$app/paths";
  import { globalStore } from "$lib/stores/global";

  const canView = $derived($globalStore.claims.includes("kpi"));
</script>

<svelte:head>
  <title>KPI Reports</title>
</svelte:head>

<main class="mx-auto max-w-5xl p-4 sm:p-6">
  <div class="mb-6">
    <p class="text-sm font-semibold tracking-wide text-blue-700 uppercase">Reports</p>
    <h1 class="text-3xl font-bold text-neutral-900">KPI Reports</h1>
    <p class="mt-2 text-neutral-600">Read-only management reports.</p>
  </div>

  {#if canView}
    <a
      class="block rounded-xl border border-neutral-200 bg-white p-5 shadow-sm transition hover:border-blue-300 hover:shadow-md"
      href={resolve("/reports/kpi/employee-branch-hours")}
    >
      <div class="flex items-start justify-between gap-4">
        <div>
          <h2 class="text-lg font-semibold text-neutral-900">Employee Branch Hours</h2>
          <p class="mt-1 text-sm text-neutral-600">
            Compare job and no-job regular hours for each employee and branch.
          </p>
        </div>
        <span class="text-2xl text-blue-600" aria-hidden="true">→</span>
      </div>
    </a>
  {:else}
    <div class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-amber-900">
      You need the kpi claim to view these reports.
    </div>
  {/if}
</main>
