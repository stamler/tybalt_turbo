<script lang="ts">
  import { onMount } from "svelte";
  import { pb } from "$lib/pocketbase";
  import type { POApprovalLimitField, POApprovalLimitsResponse } from "$lib/poApprovalLimits";

  import DSActionButton from "$lib/components/DSActionButton.svelte";
  import TimeSummaryHelp from "$lib/components/jobs/TimeSummaryHelp.svelte";

  let data = $state<POApprovalLimitsResponse | null>(null);
  let loading = $state(true);
  let loadError = $state("");
  let search = $state("");
  let division = $state("");
  let sortColumn = $state<"name" | "divisions" | POApprovalLimitField>("name");
  let ascending = $state(true);
  let controller: AbortController | undefined;

  const money = new Intl.NumberFormat("en-CA", { style: "currency", currency: "CAD" });

  async function load() {
    controller?.abort();
    const request = new AbortController();
    controller = request;
    loading = true;
    loadError = "";
    data = null;
    try {
      const response = await pb.send<POApprovalLimitsResponse>(
        "/api/purchase_orders/approval_limits",
        {
          method: "GET",
          signal: request.signal,
          requestKey: null,
        },
      );
      if (!request.signal.aborted) data = response;
    } catch (error) {
      if (!request.signal.aborted) {
        const status = (error as { status?: number }).status;
        loadError =
          status === 403 || status === 401
            ? "You do not have permission to view PO approval limits."
            : "Could not load PO approval limits. Try again.";
      }
    } finally {
      if (!request.signal.aborted) loading = false;
    }
  }

  onMount(() => {
    void load();
    return () => controller?.abort();
  });

  const divisionLabels = $derived(
    new Map(
      data?.divisions.map((d) => [d.id, `${d.code} — ${d.name}${d.active ? "" : " (inactive)"}`]) ??
        [],
    ),
  );
  const divisionCodes = $derived(new Map(data?.divisions.map((d) => [d.id, d.code]) ?? []));
  const rows = $derived.by(() => {
    const term = search.trim().toLocaleLowerCase();
    return (data?.items ?? [])
      .filter((row) => `${row.given_name} ${row.surname}`.toLocaleLowerCase().includes(term))
      .filter(
        (row) =>
          !division ||
          (row.configured && (row.divisions.length === 0 || row.divisions.includes(division))),
      )
      .sort((a, b) => {
        const names = `${a.surname} ${a.given_name}`.localeCompare(`${b.surname} ${b.given_name}`);
        if (sortColumn === "name") return ascending ? names : -names;
        // Keep missing settings last in both sort directions.
        if (a.configured !== b.configured) return a.configured ? -1 : 1;
        const comparison =
          sortColumn === "divisions"
            ? divisionSortLabel(a.divisions).localeCompare(divisionSortLabel(b.divisions))
            : a[sortColumn] - b[sortColumn];
        return (ascending ? comparison : -comparison) || names;
      });
  });

  function divisionSortLabel(ids: string[]) {
    return ids.length === 0
      ? "All divisions"
      : ids
          .map((id) => divisionCodes.get(id) || id)
          .sort()
          .join(", ");
  }

  function sort(column: "name" | "divisions" | POApprovalLimitField) {
    ascending = column === sortColumn ? !ascending : column === "name" || column === "divisions";
    sortColumn = column;
  }
</script>

<div>
  {#if loading}
    <p role="status" class="p-4 text-neutral-600">Loading PO approval limits…</p>
  {:else if loadError}
    <div role="alert" class="rounded-sm border border-red-200 bg-red-50 p-4">
      <p class="text-red-800">{loadError}</p>
      <div class="mt-2"><DSActionButton action={load} color="neutral">Retry</DSActionButton></div>
    </div>
  {:else if data}
    <!-- Match the list toolbar. Keep all three controls the same height. -->
    <div class="flex flex-wrap items-center gap-2 bg-neutral-200 p-2">
      <input
        type="search"
        aria-label="Search"
        placeholder="Search by name…"
        bind:value={search}
        class="h-9 min-w-0 flex-1 basis-64 rounded-sm border border-neutral-300 bg-white px-2 text-base"
      />
      <p role="status" class="order-3 whitespace-nowrap sm:order-none">
        {rows.length} of {data.items.length} approvers
      </p>
      <div class="relative order-2 min-w-0 flex-1 basis-48 sm:order-none sm:max-w-72">
        <!-- Inset the arrow without changing the native select behavior. -->
        <select
          aria-label="Division"
          bind:value={division}
          class="h-9 w-full appearance-none rounded-sm border border-neutral-300 bg-white pr-8 pl-2 text-base"
        >
          <option value="">All divisions</option>
          {#each data.divisions as d (d.id)}
            <option value={d.id}>{divisionLabels.get(d.id)}</option>
          {/each}
        </select>
        <svg
          aria-hidden="true"
          class="pointer-events-none absolute top-1/2 right-2 h-4 w-4 -translate-y-1/2 text-neutral-600"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"><path d="m6 9 6 6 6-6" /></svg
        >
      </div>
      <button
        type="button"
        onclick={load}
        class="order-1 h-9 rounded-sm border border-neutral-300 bg-white px-3 text-base hover:bg-neutral-100 sm:order-none"
        >Refresh</button
      >
    </div>

    <div
      class="flex flex-wrap items-center justify-between gap-x-3 border-b border-blue-300 bg-blue-200 px-2 text-neutral-700"
    >
      <h1>PO Approvers</h1>
      <div class="flex items-center gap-2 text-sm">
        <span>Maximum approval · CAD</span>
        <TimeSummaryHelp label="Help" title="How limits apply">
          <div class="mt-2 space-y-2">
            <p>
              Each amount is the person's final approval limit for that spending category. $0.00
              means no approval authority in that category. “Not configured” means the approval
              settings are missing.
            </p>
            <p>
              Division restrictions apply to every category. An empty division assignment means all
              divisions. The division filter also includes these approvers. Point to a division code
              for its name, or use the division filter to select it by name.
            </p>
            <p>
              Turbo uses the PO approval total in CAD. For recurring POs, this includes all periods.
              Some POs require two approval steps: a person with a lower nonzero limit can review
              the first step, then a person with a sufficient limit gives final approval.
            </p>
            <p>
              A second approval is required when the approval total exceeds the category threshold
              below. “None” means that category has no second-approval threshold.
            </p>
            <ul class="list-inside list-disc">
              {#each data.categories as category (category.id)}
                <li>
                  {category.label}: {category.second_approval_threshold > 0
                    ? money.format(category.second_approval_threshold)
                    : "None"}
                </li>
              {/each}
            </ul>
          </div>
        </TimeSummaryHelp>
      </div>
    </div>

    {#if data.items.length === 0}
      <p class="p-4 text-neutral-600">No active PO approvers found.</p>
    {:else if rows.length === 0}
      <p class="p-4 text-neutral-600">No approvers match your search and division.</p>
    {:else}
      <!-- Keep names visible when the amount columns scroll sideways. -->
      <!-- svelte-ignore a11y_no_noninteractive_tabindex (Keyboard users need to focus this region to scroll the table.) -->
      <div class="overflow-x-auto" role="region" aria-label="PO approval limits" tabindex="0">
        <table class="w-full text-sm">
          <caption class="sr-only"
            >PO approval limits by person and spending category, in CAD</caption
          >
          <thead class="bg-neutral-100">
            <tr>
              <th
                scope="col"
                class="sticky left-0 z-10 min-w-40 bg-neutral-100 px-3 py-3 text-left"
                aria-sort={sortColumn === "name"
                  ? ascending
                    ? "ascending"
                    : "descending"
                  : "none"}
              >
                <button
                  type="button"
                  onclick={() => sort("name")}
                  class="whitespace-nowrap hover:underline"
                  >Name {sortColumn === "name" ? (ascending ? "↑" : "↓") : ""}</button
                >
              </th>
              {#each data.categories as category (category.id)}
                <th
                  scope="col"
                  class="px-3 py-3 text-right"
                  aria-sort={sortColumn === category.limit_field
                    ? ascending
                      ? "ascending"
                      : "descending"
                    : "none"}
                >
                  <button
                    type="button"
                    onclick={() => sort(category.limit_field)}
                    class="hover:underline"
                    >{category.label}
                    {sortColumn === category.limit_field ? (ascending ? "↑" : "↓") : ""}</button
                  >
                </th>
              {/each}
              <th
                scope="col"
                class="min-w-40 px-3 py-3 text-left"
                aria-sort={sortColumn === "divisions"
                  ? ascending
                    ? "ascending"
                    : "descending"
                  : "none"}
              >
                <button type="button" onclick={() => sort("divisions")} class="hover:underline">
                  Divisions {sortColumn === "divisions" ? (ascending ? "↑" : "↓") : ""}
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            {#each rows as row (row.id)}
              <tr class="odd:bg-neutral-200 even:bg-neutral-100">
                <th
                  scope="row"
                  class="sticky left-0 z-10 max-w-52 min-w-40 bg-inherit px-3 py-3 text-left font-semibold"
                  >{row.given_name} {row.surname}</th
                >
                {#each data.categories as category (category.id)}
                  <td
                    class="px-3 py-3 text-right whitespace-nowrap tabular-nums"
                    class:text-neutral-600={row.configured && row[category.limit_field] === 0}
                    >{row.configured
                      ? money.format(row[category.limit_field])
                      : "Not configured"}</td
                  >
                {/each}
                <td class="px-3 py-3">
                  {#if !row.configured}
                    Not configured
                  {:else if row.divisions.length === 0}
                    All divisions
                  {:else}
                    <ul class="flex max-w-56 flex-wrap gap-1">
                      {#each row.divisions as id (id)}<li>
                          <abbr
                            title={divisionLabels.get(id) ?? `Unknown division (${id})`}
                            class="cursor-help rounded-sm bg-neutral-100 px-1.5 py-0.5 no-underline"
                            >{divisionCodes.get(id) || id}</abbr
                          >
                        </li>{/each}
                    </ul>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {/if}
</div>
