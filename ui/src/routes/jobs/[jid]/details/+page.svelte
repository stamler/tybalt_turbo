<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import type { PageData } from "./$types";
  import { onMount } from "svelte";
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";

  export let data: PageData;
  // Use data.job directly (Svelte 5 without $:)

  function personName(person: any) {
    if (!person) return "";
    return `${person.given_name || person.name || ""} ${person.surname || ""}`.trim();
  }

  // ---------------------------------------------------------------------------
  // Time summary support

  interface SummaryRow {
    total_hours: number | string;
    earliest_entry?: string;
    latest_entry?: string;
    divisions?: any[];
    time_types?: any[];
    names?: any[];
    categories?: any[];
  }

  // Holds the current aggregated summary returned from the API
  let summary: SummaryRow = {
    total_hours: 0,
    divisions: [],
    time_types: [],
    names: [],
    categories: [],
  };

  // Active filters – only one of each type can be active at a time
  let selectedDivision: { id: string; code: string } | null = null;
  let selectedTimeType: { id: string; code: string } | null = null;
  let selectedName: { id: string; name: string } | null = null;
  let selectedCategory: { id: string; name: string } | null = null;

  let loading = false;

  // Parse helper since some fields come back as json strings
  const parseArr = (val: any): any[] => {
    if (!val) return [];
    if (Array.isArray(val)) return val as any[];
    try {
      return JSON.parse(val as string);
    } catch {
      return [];
    }
  };

  async function fetchSummary() {
    loading = true;
    try {
      const params = new URLSearchParams();
      if (selectedDivision) params.set("division", selectedDivision.id);
      if (selectedTimeType) params.set("time_type", selectedTimeType.id);
      if (selectedName) params.set("uid", selectedName.id);
      if (selectedCategory) params.set("category", selectedCategory.id);

      const query = params.toString();
      const res: any = await pb.send(
        `/api/jobs/${data.job.id}/time/summary${query ? "?" + query : ""}`,
        {
          method: "GET",
        },
      );

      summary = {
        total_hours: res.total_hours ?? 0,
        earliest_entry: res.earliest_entry ?? "",
        latest_entry: res.latest_entry ?? "",
        divisions: parseArr(res.divisions),
        time_types: parseArr(res.time_types),
        names: parseArr(res.names),
        categories: parseArr(res.categories),
      };
    } catch (err) {
      console.error("Failed to fetch time summary", err);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchSummary();
    fetchEntries(true);
  });

  // Utility to toggle a filter. Passing null clears the filter.
  function toggleFilter(type: "division" | "time_type" | "name" | "category", value: any) {
    switch (type) {
      case "division":
        selectedDivision = selectedDivision && selectedDivision.id === value.id ? null : value;
        break;
      case "time_type":
        selectedTimeType = selectedTimeType && selectedTimeType.id === value.id ? null : value;
        break;
      case "name":
        selectedName = selectedName && selectedName.id === value.id ? null : value;
        break;
      case "category":
        selectedCategory = selectedCategory && selectedCategory.id === value.id ? null : value;
        break;
    }
    fetchSummary();
    fetchEntries(true);
  }

  function clearFilter(type: "division" | "time_type" | "name" | "category") {
    if (type === "division") selectedDivision = null;
    if (type === "time_type") selectedTimeType = null;
    if (type === "name") selectedName = null;
    if (type === "category") selectedCategory = null;
    fetchSummary();
    fetchEntries(true);
  }

  // ---------------------------------------------------------------------------
  // Paginated entries support
  interface JobTimeEntry {
    id: string;
    date: string;
    hours: number;
    description: string;
    work_record: string;
    week_ending: string;
    tsid: string;
    division_code: string;
    time_type_code: string;
    surname: string;
    given_name: string;
    category_name: string;
  }

  let entries: JobTimeEntry[] = [];
  let entriesPage = 1;
  let entriesLimit = 50;
  let entriesTotalPages = 1;
  let entriesLoading = false;

  // resets and fetches first page with current filters
  async function fetchEntries(reset = false) {
    if (reset) {
      entriesPage = 1;
      entries = [];
    }
    entriesLoading = true;
    try {
      const params = new URLSearchParams();
      params.set("page", entriesPage.toString());
      params.set("limit", entriesLimit.toString());
      if (selectedDivision) params.set("division", selectedDivision.id);
      if (selectedTimeType) params.set("time_type", selectedTimeType.id);
      if (selectedName) params.set("uid", selectedName.id);
      if (selectedCategory) params.set("category", selectedCategory.id);
      const query = params.toString();
      const res: any = await pb.send(`/api/jobs/${data.job.id}/time/entries?${query}`, {
        method: "GET",
      });
      if (Array.isArray(res.data)) {
        if (reset) entries = res.data;
        else entries = [...entries, ...res.data];
      }
      entriesTotalPages = res.total_pages || 1;
    } catch (err) {
      console.error("Failed to fetch time entries", err);
    } finally {
      entriesLoading = false;
    }
  }

  // load more pages
  function loadMore() {
    if (entriesPage < entriesTotalPages) {
      entriesPage += 1;
      fetchEntries(false);
    }
  }
</script>

<div class="mx-auto space-y-4 p-4">
  <h1 class="text-2xl font-bold">Job Details</h1>

  <div class="space-y-2 rounded bg-neutral-100 p-4">
    <div class="flex items-center gap-2">
      <span class="font-semibold">Job Number:</span>
      <span>{data.job.number}</span>
      {#if data.job.number?.startsWith("P")}
        <DsLabel color="yellow">proposal</DsLabel>
      {/if}
    </div>
    <div><span class="font-semibold">Description:</span> {data.job.description}</div>
    {#if data.job.status}
      <div><span class="font-semibold">Status:</span> {data.job.status}</div>
    {/if}

    {#if data.job.client}
      <div>
        <span class="font-semibold">Client:</span>
        <a href={`/clients/${data.job.client.id}/details`} class="text-blue-600 hover:underline">
          {data.job.client.name}
        </a>
      </div>
    {/if}

    {#if data.job.contact && (data.job.contact.given_name || data.job.contact.surname)}
      <div><span class="font-semibold">Contact:</span> {personName(data.job.contact)}</div>
    {/if}

    {#if data.job.manager && (data.job.manager.given_name || data.job.manager.surname)}
      <div><span class="font-semibold">Manager:</span> {personName(data.job.manager)}</div>
    {/if}

    {#if data.job.alternate_manager && (data.job.alternate_manager.given_name || data.job.alternate_manager.surname)}
      <div>
        <span class="font-semibold">Alternate Manager:</span>
        {personName(data.job.alternate_manager)}
      </div>
    {/if}

    {#if data.job.job_owner && (data.job.job_owner.given_name || data.job.job_owner.surname)}
      <div><span class="font-semibold">Job Owner:</span> {personName(data.job.job_owner)}</div>
    {/if}

    {#if data.job.proposal_id}
      <div>
        <span class="font-semibold">Proposal:</span>
        <a href={`/jobs/${data.job.proposal_id}/details`} class="text-blue-600 hover:underline">
          {data.job.proposal_number || data.job.proposal_id}
        </a>
      </div>
    {/if}

    <div>
      <span class="font-semibold">FN Agreement:</span>
      {data.job.fn_agreement ? "Yes" : "No"}
    </div>

    {#if data.job.project_award_date}
      <div>
        <span class="font-semibold">Project Award Date:</span>
        {data.job.project_award_date}
      </div>
    {/if}

    {#if data.job.proposal_opening_date}
      <div>
        <span class="font-semibold">Proposal Opening Date:</span>
        {data.job.proposal_opening_date}
      </div>
    {/if}

    {#if data.job.proposal_submission_due_date}
      <div>
        <span class="font-semibold">Proposal Submission Due:</span>
        {data.job.proposal_submission_due_date}
      </div>
    {/if}

    {#if data.job.divisions && Array.isArray(data.job.divisions)}
      <div>
        <span class="font-semibold">Divisions:</span>
        {#each data.job.divisions as division, idx}
          {division.name} ({division.code}){idx < data.job.divisions.length - 1 ? ", " : ""}
        {/each}
      </div>
    {/if}

    {#if data.job.projects && data.job.projects.length > 0}
      <div>
        <span class="font-semibold">Projects:</span>
        {#each data.job.projects as p, i}
          <a href={`/jobs/${p.id}/details`} class="text-blue-600 hover:underline">{p.number}</a>{i <
          data.job.projects.length - 1
            ? ", "
            : ""}
        {/each}
      </div>
    {/if}
  </div>

  <div class="flex gap-2">
    <DsActionButton
      action={`/jobs/${data.job.id}/edit`}
      icon="mdi:pencil"
      title="Edit Job"
      color="blue"
    />
  </div>

  <!-- Time Section (Summary | Entries) -->
  <div class="space-y-4 rounded bg-neutral-50 p-4 shadow-sm">
    <h2 class="text-xl font-bold">Time</h2>

    {#if loading}
      <div>Loading…</div>
    {:else}
      <!-- Summary strip -->
      <!-- Aggregates -->
      <div class="space-y-1">
        <div><span class="font-semibold">Total Hours:</span> {summary.total_hours}</div>
        {#if summary.earliest_entry}
          <div>
            <span class="font-semibold">Date Range:</span>
            {summary.earliest_entry} – {summary.latest_entry}
          </div>
        {/if}
      </div>

      <!-- Filter chips row -->
      {#if summary.divisions || summary.time_types || summary.names || summary.categories}
        <div class="flex flex-wrap gap-2 pt-2">
          <!-- Division chip list -->
          {#if summary.divisions && summary.divisions.length > 0}
            <span class="font-semibold">Divisions:</span>
            {#each summary.divisions as d}
              <button on:click={() => toggleFilter("division", d)} class="focus:outline-none">
                <DsLabel color="blue" style={selectedDivision?.id === d.id ? "inverted" : undefined}
                  >{d.code}</DsLabel
                >
              </button>
            {/each}
          {/if}

          <!-- Time type chips -->
          {#if summary.time_types && summary.time_types.length > 0}
            <span class="font-semibold">Time Types:</span>
            {#each summary.time_types as tt}
              <button on:click={() => toggleFilter("time_type", tt)} class="focus:outline-none">
                <DsLabel
                  color="green"
                  style={selectedTimeType?.id === tt.id ? "inverted" : undefined}>{tt.code}</DsLabel
                >
              </button>
            {/each}
          {/if}

          <!-- Staff chips -->
          {#if summary.names && summary.names.length > 0}
            <span class="font-semibold">Staff:</span>
            {#each summary.names as n}
              <button on:click={() => toggleFilter("name", n)} class="focus:outline-none">
                <DsLabel color="purple" style={selectedName?.id === n.id ? "inverted" : undefined}
                  >{n.name}</DsLabel
                >
              </button>
            {/each}
          {/if}

          <!-- Category chips -->
          {#if summary.categories && summary.categories.length > 0}
            <span class="font-semibold">Categories:</span>
            {#each summary.categories as c}
              <button on:click={() => toggleFilter("category", c)} class="focus:outline-none">
                <DsLabel color="red" style={selectedCategory?.id === c.id ? "inverted" : undefined}
                  >{c.name}</DsLabel
                >
              </button>
            {/each}
          {/if}
        </div>
      {/if}
    {/if}

    <!-- Entries list always visible -->
    {#if entriesLoading && entries.length === 0}
      <div>Loading…</div>
    {:else if entries.length === 0}
      <div>No entries found.</div>
    {:else}
      <div class="overflow-hidden rounded-lg">
        <DsList items={entries} search={false} inListHeader="Time Entries">
          {#snippet anchor(item: JobTimeEntry)}{item.date}{/snippet}
          {#snippet headline(item: JobTimeEntry)}
            {#if item.time_type_code === "R"}
              <span>{item.division_code}</span>
            {:else}
              <span>{item.time_type_code}</span>
            {/if}
          {/snippet}
          {#snippet byline(item: JobTimeEntry)}{item.hours}h{/snippet}
          {#snippet line1(item: JobTimeEntry)}{item.given_name} {item.surname}{/snippet}
          {#snippet line2(item: JobTimeEntry)}{item.description}{/snippet}
        </DsList>
        {#if entriesPage < entriesTotalPages}
          <div class="mt-4 text-center">
            <button
              class="rounded bg-blue-600 px-4 py-2 text-white"
              on:click={loadMore}
              disabled={entriesLoading}
            >
              {entriesLoading ? "Loading…" : "Load More"}
            </button>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>
