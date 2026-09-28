<script lang="ts">
  import Page from "../../src/routes/pos/approval-limits/+page.svelte";
  import { navSections } from "$lib/navConfig";
  import { globalStore } from "$lib/stores/global";
  import type { Writable } from "svelte/store";

  // The browser harness replaces the application store with a writable stub.
  const testStore = globalStore as unknown as Writable<{ claims: string[] }>;

  let access = $state("report");
  $effect(() => {
    testStore.update((state) => ({ ...state, claims: access === "none" ? [] : access.split(",") }));
  });
</script>

<label
  >Test access
  <select aria-label="Test access" bind:value={access}>
    <option value="report">Report</option>
    <option value="admin">Admin</option>
    <option value="report,admin">Report and admin</option>
    <option value="none">None</option>
  </select>
</label>
<Page />

<output hidden data-testid="nav-config">{JSON.stringify(navSections)}</output>
