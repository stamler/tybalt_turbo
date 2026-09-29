<script lang="ts">
  import { SvelteMap } from "svelte/reactivity";
  import MiniSearch from "minisearch";
  import { pb } from "$lib/pocketbase";
  import DSAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { rankWIP, type WIPMode, type WIPReportData } from "$lib/reports/wipReports";
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
  let requireTime = $state(true);
  let includeExpenses = $state(true);
  let includePOs = $state(true);
  let data = $state<WIPReportData | null>(null);
  let error = $state("");
  let retry = $state(0);
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
    const rows = rankWIP(data?.items ?? [], includeExpenses, includePOs);
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
    if (!permitted || (reportMode === "division" && !selectedDivision)) return;
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
<main class="mx-auto max-w-7xl space-y-5 p-4 sm:p-6">
  <div class="flex items-start justify-between gap-3">
    <div>
      <h1 class="text-3xl font-bold">{title}</h1>
      <p class="mt-2 text-sm text-neutral-600">Active projects · highest percentage first · CAD</p>
    </div>
    <WIPHelp />
  </div>
  {#if !allowed}
    <p role="alert">
      You need to be a branch manager or hold the kpi or admin claim to view this report.
    </p>
  {:else}
    {#if mode === "branch"}
      <label class="block max-w-sm font-medium"
        >Branch
        <select
          class="mt-1 block w-full rounded-sm border border-neutral-300 p-2"
          aria-label="Branch"
          bind:value={branch}
        >
          <option value="">All branches</option>
          {#each branches as item (item.id)}<option value={item.id}>{item.name}</option>{/each}
        </select>
      </label>
    {:else if mode === "division"}
      <div class="max-w-md">
        <DSAutoComplete
          bind:value={division}
          index={divisionIndex}
          errors={{}}
          fieldName="division"
          uiName="Division"
        >
          {#snippet resultTemplate(item)}{item.code} - {item.name}{/snippet}
        </DSAutoComplete>
      </div>
      <label class="flex items-center gap-2"
        ><input type="checkbox" bind:checked={requireTime} />Only jobs with recorded work hours in
        this division</label
      >
      <p class="text-sm text-neutral-600">
        Each row shows the whole job’s WIP across all divisions.
      </p>
    {/if}
    <WIPControls bind:includeExpenses bind:includePOs />
    {#if error}
      <div role="alert" class="flex items-center gap-3 text-red-700">
        <p>{error}</p>
        <button class="rounded-sm border px-3 py-2" onclick={() => retry++}>Try again</button>
      </div>
    {:else if mode === "division" && !division}
      <p role="status">Select a division to view its projects.</p>
    {:else if data}
      <p class="text-sm text-neutral-600">Job to date · {data.as_of}</p>
      {#if data.excluded_project_value > 0}<p class="text-sm text-neutral-600">
          {data.excluded_project_value} active projects excluded: no project value greater than zero.
        </p>{/if}
      {#if data.items.length === 0}
        <p role="status">
          {mode === "my"
            ? "No projects to show. You are not the manager of any active projects with a project value greater than zero."
            : "No active projects with a project value greater than zero match these filters."}
        </p>
      {/if}
      {#each groups as [id, group] (id)}
        {@const ranked = group.rows.filter((row) => row.view.percent !== null)}
        {@const partial = group.rows.filter((row) => row.view.percent === null)}
        <section class="space-y-3" aria-label={group.name}>
          {#if mode === "branch"}<h2 class="text-xl font-semibold">{group.name}</h2>{/if}
          {#if ranked.length}<WIPReportTable
              rows={ranked}
              title={`${group.name}: ranked WIP`}
            />{/if}
          {#if partial.length}
            <h3 class="font-semibold text-amber-800">Cannot rank ({partial.length})</h3>
            <p class="text-sm text-neutral-600">
              Some included amounts cannot be priced. Known values are shown; overall percentages
              are unavailable.
            </p>
            <WIPReportTable rows={partial} title={`${group.name}: cannot rank`} />
          {/if}
        </section>
      {/each}
    {:else}<p role="status">Loading WIP report…</p>{/if}
  {/if}
</main>
