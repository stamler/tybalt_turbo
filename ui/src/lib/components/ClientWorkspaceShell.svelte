<script lang="ts">
  import type { Snippet } from "svelte";
  import DSTabBar from "./DSTabBar.svelte";
  import { navigating, page } from "$app/state";
  import {
    workspaceHref,
    safeReturnTo,
    currentWorkspaceUrl,
    type ClientWorkspaceTab,
  } from "$lib/clientWorkspace";
  import { safeWorkflowReturn } from "$lib/clientWorkflowNavigation";

  let {
    clientId,
    clientName,
    activeTab = "details",
    children,
  }: {
    clientId: string;
    clientName: string;
    activeTab?: ClientWorkspaceTab;
    children: Snippet;
  } = $props();
  const source = $derived(
    ["details", "jobs", "notes"].some((tab) => page.url.pathname === `/clients/${clientId}/${tab}`)
      ? currentWorkspaceUrl(clientId, page.url, page.state)
      : new URL(safeReturnTo(clientId, page.url.searchParams.get("return_to")), page.url),
  );
  const workflowReturn = $derived(safeWorkflowReturn(source.searchParams.get("workflow_return")));
  // Show progress while another page of this client's workspace loads.
  const loading = $derived(!!navigating.to?.url.pathname.startsWith(`/clients/${clientId}/`));
  function href(tab: ClientWorkspaceTab) {
    return workspaceHref(clientId, source.searchParams, { edit: null }, tab);
  }
</script>

<div class="mx-auto min-w-0 space-y-4 p-4">
  <h1 class="text-2xl font-bold">{clientName}</h1>
  {#if workflowReturn}
    <p class="rounded-sm border border-yellow-300 bg-yellow-50 p-2">
      You are updating this client from a job form with unsaved changes.
      <a href={workflowReturn} class="font-semibold text-blue-700 underline"
        >Return to the job form</a
      >
    </p>
  {/if}
  <div class="flex items-center gap-3 overflow-x-auto">
    <DSTabBar
      tabs={[
        { label: "Client details", href: href("details"), active: activeTab === "details" },
        { label: "Jobs", href: href("jobs"), active: activeTab === "jobs" },
        { label: "Notes", href: href("notes"), active: activeTab === "notes" },
      ]}
    />
    {#if loading}<span role="status" class="shrink-0 text-sm text-neutral-600">Loading…</span>{/if}
  </div>
  <div class="transition-opacity {loading ? 'opacity-50' : ''}" aria-busy={loading}>
    {@render children()}
  </div>
</div>
