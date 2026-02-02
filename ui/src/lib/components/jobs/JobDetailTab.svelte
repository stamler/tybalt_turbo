<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsLabel from "../DsLabel.svelte";
  import type { FilterDef } from "./types";

  // TYPES
  interface $$Props {
    active?: boolean;
    jobId: string;
    summaryUrl: string;
    listUrl: string;
    filterDefs?: FilterDef[];
    children: (args: {
      summary: Record<string, any>;
      items: any[];
      listLoading: boolean;
      loadMore: () => void;
      page: number;
      totalPages: number;
    }) => any;
  }

  // PROPS
  let { active = false, jobId, summaryUrl, listUrl, filterDefs = [], children } = $props();

  // ---------------------------------------------------------------------------
  // STATE

  // Summary
  let summary: Record<string, any> = $state({});
  let summaryLoading = $state(false);

  // List
  let items: any[] = $state([]);
  let page = $state(1);
  let limit = 50;
  let totalPages = $state(1);
  let listLoading = $state(false);

  // Filters
  let selectedFilters: Record<string, any> = $state({});

  let initialized = $state(false);

  // ---------------------------------------------------------------------------
  // LIFECYCLE & REACTIONS

  $effect(() => {
    if (active && !initialized) {
      initialized = true;
      fetchSummary();
      fetchList(true);
    }
  });

  // ---------------------------------------------------------------------------
  // HELPERS
  const parseArr = (val: any): any[] => {
    if (!val) return [];
    if (Array.isArray(val)) return val as any[];
    try {
      return JSON.parse(val as string);
    } catch {
      return [];
    }
  };

  // ---------------------------------------------------------------------------
  // DATA FETCHING

  async function fetchSummary() {
    summaryLoading = true;
    try {
      const params = new URLSearchParams();
      for (const f of filterDefs) {
        if (selectedFilters[f.type]) {
          const paramName = f.queryParam || f.type;
          params.set(paramName, selectedFilters[f.type][f.valueProperty]);
        }
      }
      const query = params.toString();
      const url = `${summaryUrl}${query ? "?" + query : ""}`;
      const res: any = await pb.send(url, { method: "GET" });

      const newSummary: Record<string, any> = { ...res };
      for (const def of filterDefs) {
        if (res[def.summaryProperty]) {
          newSummary[def.summaryProperty] = parseArr(res[def.summaryProperty]);
        }
      }
      summary = newSummary;
    } catch (err) {
      console.error(`Failed to fetch summary from ${summaryUrl}`, err);
    } finally {
      summaryLoading = false;
    }
  }

  async function fetchList(reset = false) {
    if (reset) {
      page = 1;
      items = [];
    }
    listLoading = true;
    try {
      const params = new URLSearchParams();
      params.set("page", page.toString());
      params.set("limit", limit.toString());
      for (const f of filterDefs) {
        if (selectedFilters[f.type]) {
          const paramName = f.queryParam || f.type;
          params.set(paramName, selectedFilters[f.type][f.valueProperty]);
        }
      }
      const query = params.toString();
      const url = `${listUrl}?${query}`;
      const res: any = await pb.send(url, { method: "GET" });

      if (Array.isArray(res.data)) {
        items = reset ? res.data : [...items, ...res.data];
      }
      totalPages = res.total_pages || 1;
    } catch (err) {
      console.error(`Failed to fetch list from ${listUrl}`, err);
    } finally {
      listLoading = false;
    }
  }

  function loadMore() {
    if (page < totalPages) {
      page += 1;
      fetchList(false);
    }
  }

  // ---------------------------------------------------------------------------
  // FILTERING

  function toggleFilter(type: string, value: any) {
    const def = filterDefs.find((f) => f.type === type);
    if (!def) return;

    const currentFilter = selectedFilters[type];
    if (currentFilter && currentFilter[def.valueProperty] === value[def.valueProperty]) {
      selectedFilters[type] = null;
    } else {
      selectedFilters[type] = value;
    }

    // Refetch data
    fetchSummary();
    fetchList(true);
  }
</script>

<div class="space-y-4 rounded-sm bg-neutral-50 py-4 shadow-xs">
  <!-- Summary & Filters -->
  <div class="px-4">
    {#if summaryLoading}
      <div>Loading…</div>
    {:else}
      <!-- Filter Chips -->
      <div class="flex flex-wrap gap-2 pt-2">
        {#each filterDefs as def}
          <!-- Only show filter if there's a choice to be made, or if a filter is already active -->
          {#if (summary[def.summaryProperty] && summary[def.summaryProperty].length > 1) || selectedFilters[def.type]}
            <span class="font-semibold">{def.label}:</span>
            {#each summary[def.summaryProperty] as item}
              <button onclick={() => toggleFilter(def.type, item)} class="focus:outline-hidden">
                <DsLabel
                  color={def.color}
                  style={selectedFilters[def.type]?.[def.valueProperty] === item[def.valueProperty]
                    ? "inverted"
                    : undefined}
                >
                  {item[def.displayProperty]}
                </DsLabel>
              </button>
            {/each}
          {/if}
        {/each}
      </div>
    {/if}
  </div>

  <!-- Default Slot -->
  {@render children({ summary, items, listLoading, loadMore, page, totalPages })}
</div>
