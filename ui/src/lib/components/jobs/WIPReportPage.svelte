<script lang="ts">
  import { authStore } from "$lib/stores/auth";
  import { globalStore } from "$lib/stores/global";
  import { pb } from "$lib/pocketbase";
  import { branches } from "$lib/stores/branches";
  import { divisions } from "$lib/stores/divisions";
  import { canViewWIPReports, type WIPMode } from "$lib/reports/wipReports";
  import WIPReport from "./WIPReport.svelte";
  let { mode }: { mode: WIPMode } = $props();
  const userId = $derived($authStore?.model?.id ?? "");
  const managedBranch = $derived($branches.items.find((branch) => branch.manager === userId));
  const allowed = $derived(
    mode === "my" || canViewWIPReports($globalStore.claims, !!managedBranch),
  );
  let defaultDivision = $state("");
  let defaultsLoading = $state(false);
  let defaultsError = $state(false);
  let retry = $state(0);
  const requestId = $props.id();
  $effect(() => {
    void retry;
    void userId;
    defaultsError = false;
    defaultDivision = "";
    defaultsLoading = mode === "division";
    if (mode !== "division") return;
    let current = true;
    const controller = new AbortController();
    pb.send<{ default_division: string }>("/api/users/defaults", {
      method: "GET",
      requestKey: requestId,
      signal: controller.signal,
    })
      .then((result) => {
        if (current) defaultDivision = result.default_division;
      })
      .catch(() => {
        if (current) defaultsError = true;
      })
      .finally(() => {
        if (current) defaultsLoading = false;
      });
    return () => {
      current = false;
      controller.abort();
    };
  });
</script>

{#if mode !== "my" && (!$branches.initialized || (mode === "division" && (!$divisions.initialized || defaultsLoading || defaultsError)))}
  <main class="p-6">
    {#if $branches.error || $divisions.error || defaultsError}
      <p role="alert">Failed to load report selections.</p>
      <button
        class="mt-3 rounded-sm border px-3 py-2"
        onclick={() => {
          void branches.init();
          void divisions.init();
          retry++;
        }}>Try again</button
      >
    {:else}<p role="status">Loading report selections…</p>{/if}
  </main>
{:else}
  {#key `${mode}:${userId}`}
    <WIPReport
      {mode}
      {allowed}
      branches={$branches.items}
      divisions={$divisions.items}
      initialBranch={managedBranch?.id ?? ""}
      initialDivision={defaultDivision}
    />
  {/key}
{/if}
