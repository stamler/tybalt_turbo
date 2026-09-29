<script lang="ts">
  import { onMount } from "svelte";
  import { SvelteMap } from "svelte/reactivity";
  import MiniSearch from "minisearch";
  import { pb } from "$lib/pocketbase";
  import DSAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import {
    filterWIP,
    rankWIP,
    type WIPMode,
    type WIPReportData,
    type WIPSort,
  } from "$lib/reports/wipReports";
  import WIPControls from "./WIPControls.svelte";
  import WIPHelp from "./WIPHelp.svelte";
  import WIPReportTable from "./WIPReportTable.svelte";

  let {
    mode,
    allowed = true,
    branches = [],
    divisions = [],
    initialBranch = "",
    initialDivision = "",
  }: {
    mode: WIPMode;
    allowed?: boolean;
    branches?: { id: string; name: string }[];
    divisions?: { id: string; code: string; name: string }[];
    initialBranch?: string;
    initialDivision?: string;
  } = $props();
  // Defaults apply when the page opens. Later refreshes must not replace a selection.
  // svelte-ignore state_referenced_locally
  let branch = $state(initialBranch);
  // svelte-ignore state_referenced_locally
  let division = $state(initialDivision);
  let selectionReady = $state(false);
  let selectionError = $state("");

  function readSelection() {
    if (mode === "my") return;
    const linked = new URLSearchParams(window.location.hash.slice(1)).get(mode);
    const selected = linked ?? (mode === "branch" ? initialBranch : initialDivision);
    const choices = mode === "branch" ? branches : divisions;
    const valid = !selected || choices.some((item) => item.id === selected);
    selectionError = valid ? "" : `The linked ${mode} is not available. Select a ${mode}.`;
    if (mode === "branch") branch = selected;
    else division = valid ? selected : "";
  }

  function select(value: string) {
    selectionError = "";
    if (mode === "branch") branch = value;
    else division = value;
    // Keep an empty value explicit so reload does not restore the user's default.
    window.location.hash = new URLSearchParams({ [mode]: value }).toString();
  }

  onMount(() => {
    readSelection();
    selectionReady = true;
    window.addEventListener("hashchange", readSelection);
    return () => window.removeEventListener("hashchange", readSelection);
  });
  const divisionCode = $derived(divisions.find((item) => item.id === division)?.code);
  let requireTime = $state(true);
  let includeExpenses = $state(true);
  let includePOs = $state(true);
  let data = $state<WIPReportData | null>(null);
  let error = $state("");
  let retry = $state(0);
  let search = $state("");
  let sort = $state<WIPSort>("percent");
  const filtered = $derived(filterWIP(data?.items ?? [], search));
  const requestId = $props.id();
  const title = $derived(
    mode === "my" ? "My WIP" : mode === "branch" ? "Branch WIP" : "Division WIP",
  );
  const divisionIndex = $derived.by(() => {
    const index = new MiniSearch<{ id: string; code: string; name: string }>({
      fields: ["code", "name"],
      storeFields: ["id", "code", "name"],
      searchOptions: { combineWith: "AND" },
    });
    index.addAll(divisions);
    return index;
  });
  const groups = $derived.by(() => {
    const rows = rankWIP(filtered, includeExpenses, includePOs, sort);
    const groups = new SvelteMap<string, { name: string; rows: typeof rows }>();
    for (const row of rows) {
      const id = mode === "branch" ? row.data.branch_id : "all";
      if (!groups.has(id))
        groups.set(id, {
          name: mode === "branch" ? row.data.branch || "No branch" : "Projects",
          rows: [],
        });
      groups.get(id)!.rows.push(row);
    }
    return [...groups.entries()].sort((a, b) => a[1].name.localeCompare(b[1].name));
  });

  $effect(() => {
    const reportMode = mode;
    const permitted = allowed;
    const selectedBranch = branch;
    const selectedDivision = division;
    const needsTime = requireTime;
    void retry;
    data = null;
    error = "";
    if (
      !selectionReady ||
      selectionError ||
      !permitted ||
      (reportMode === "division" && !selectedDivision)
    )
      return;
    let current = true;
    const controller = new AbortController();
    pb.send<WIPReportData>(`/api/wip/${reportMode}`, {
      method: "GET",
      requestKey: requestId,
      signal: controller.signal,
      query:
        reportMode === "branch"
          ? { branch: selectedBranch }
          : reportMode === "division"
            ? { division: selectedDivision, require_time: needsTime }
            : {},
    })
      .then((result) => {
        if (current) data = result;
      })
      .catch((reason) => {
        if (current)
          error =
            reason?.status === 403
              ? "You do not have permission to view this report."
              : "Failed to load WIP report.";
      });
    return () => {
      current = false;
      controller.abort();
    };
  });
</script>

<svelte:head><title>{title}</title></svelte:head>
<main aria-label={title}>
  {#if !allowed}
    <p role="alert" class="p-4">
      You need to be a branch manager or hold the kpi or admin claim to view this report.
    </p>
  {:else}
    <div class="flex flex-wrap items-center gap-2 bg-neutral-200 p-2">
      <input
        type="search"
        aria-label="Search projects"
        placeholder="Search projects, clients, managers…"
        title="Search project number, client, description, manager, and branch"
        bind:value={search}
        class="h-9 min-w-0 flex-1 basis-64 rounded-sm border border-neutral-300 bg-white px-2 text-base"
      />
      {#if data}<span class="text-sm whitespace-nowrap" aria-live="polite"
          >{filtered.length} of {data.items.length} projects</span
        >{/if}
      {#if mode === "branch"}
        <span class="relative min-w-0 flex-1 basis-48 sm:max-w-72">
          <select
            aria-label="Branch"
            class="h-9 w-full appearance-none rounded-sm border border-neutral-300 bg-white pr-8 pl-3"
            bind:value={() => branch, select}
          >
            {#if selectionError}<option value={branch} disabled>Select a branch</option>{/if}
            <option value="">All branches</option>
            {#each branches as item (item.id)}<option value={item.id}>{item.name}</option>{/each}
          </select>
          <svg
            aria-hidden="true"
            class="pointer-events-none absolute top-1/2 right-2 h-4 w-4 -translate-y-1/2 text-neutral-600"
            viewBox="0 0 20 20"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path d="m6 8 4 4 4-4" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </span>
      {:else if mode === "division"}
        <div
          class="min-w-0 flex-1 basis-64 rounded-sm border border-neutral-300 bg-white px-2 py-1 sm:max-w-md"
        >
          <DSAutoComplete
            bind:value={() => division, select}
            index={divisionIndex}
            errors={{}}
            fieldName="division"
            uiName="Division"
          >
            {#snippet resultTemplate(item)}{item.code} - {item.name}{/snippet}
          </DSAutoComplete>
        </div>
      {/if}
    </div>
    <div
      class="flex flex-wrap items-center gap-x-6 gap-y-2 border-b border-neutral-300 bg-neutral-100 px-2 py-2 text-sm"
    >
      <WIPControls bind:includeExpenses bind:includePOs />
      <label class="flex items-center gap-2 whitespace-nowrap">
        Sort by
        <span class="relative">
          <select
            bind:value={sort}
            class="h-8 appearance-none rounded-sm border border-neutral-300 bg-white pr-8 pl-3"
          >
            <option value="percent">% of project · highest first</option>
            <option value="remaining">Remaining · lowest first</option>
          </select>
          <svg
            aria-hidden="true"
            class="pointer-events-none absolute top-1/2 right-2 h-4 w-4 -translate-y-1/2 text-neutral-600"
            viewBox="0 0 20 20"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path d="m6 8 4 4 4-4" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </span>
      </label>
      {#if mode === "division"}
        <label
          class="flex items-center gap-2"
          title="Require positive recorded work hours in this division. Each row still shows the whole job’s WIP."
        >
          <input type="checkbox" bind:checked={requireTime} />Must include {divisionCode ||
            "division"} hours
        </label>
      {/if}
    </div>
    <div
      class="flex flex-wrap items-center justify-between gap-x-3 border-b border-blue-300 bg-blue-200 px-2 text-neutral-700"
    >
      <h1>{title}</h1>
      <div class="flex flex-wrap items-center gap-2 text-sm">
        <span>{data ? `${data.as_of} · ` : ""}CAD</span>
        {#if data && data.excluded_project_value > 0}
          <span title="Active projects excluded because their project value is zero or missing."
            >{data.excluded_project_value} without project value</span
          >
        {/if}
        <WIPHelp />
      </div>
    </div>
    {#if selectionError}
      <p role="alert" class="p-4 text-red-700">{selectionError}</p>
    {:else if error}
      <div role="alert" class="flex items-center gap-3 p-4 text-red-700">
        <p>{error}</p>
        <button class="rounded-sm border px-3 py-2" onclick={() => retry++}>Try again</button>
      </div>
    {:else if mode === "division" && !division}
      <p role="status" class="p-4">Select a division to view its projects.</p>
    {:else if data}
      {#if data.items.length === 0}
        <p role="status" class="p-4">
          {mode === "my"
            ? "No projects to show. You are not the manager of any active projects with a project value greater than zero."
            : "No active projects with a project value greater than zero match these filters."}
        </p>
      {:else if filtered.length === 0}
        <p role="status" class="p-4">No projects match your search.</p>
      {/if}
      {#each groups as [id, group] (id)}
        {@const ranked = group.rows.filter((row) => row.view.percent !== null)}
        {@const partial = group.rows.filter((row) => row.view.percent === null)}
        <section aria-label={group.name}>
          {#if mode === "branch"}<h2
              class="border-b border-neutral-300 bg-neutral-200 px-2 py-1 text-sm font-semibold"
            >
              {group.name}
            </h2>{/if}
          {#if ranked.length}<WIPReportTable
              rows={ranked}
              title={`${group.name}: ranked WIP`}
              {includeExpenses}
              {includePOs}
            />{/if}
          {#if partial.length}
            <h2
              class="border-b border-amber-200 bg-amber-50 px-2 py-1 text-sm text-amber-900"
              title="Some included amounts cannot be priced. Known values are shown; overall percentages and remaining balances are unavailable."
            >
              Cannot rank ({partial.length}) · known values only
            </h2>
            <WIPReportTable
              rows={partial}
              title={`${group.name}: cannot rank`}
              {includeExpenses}
              {includePOs}
            />
          {/if}
        </section>
      {/each}
    {:else}<p role="status" class="p-4">Loading WIP report…</p>{/if}
  {/if}
</main>
