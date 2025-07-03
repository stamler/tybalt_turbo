<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import type { PageData } from "./$types";
  import { onMount } from "svelte";
  import { pb } from "$lib/pocketbase";

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

  onMount(fetchSummary);

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
  }

  function clearFilter(type: "division" | "time_type" | "name" | "category") {
    if (type === "division") selectedDivision = null;
    if (type === "time_type") selectedTimeType = null;
    if (type === "name") selectedName = null;
    if (type === "category") selectedCategory = null;
    fetchSummary();
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

  <!-- Time Summary Section -->
  <div class="space-y-4 rounded bg-neutral-50 p-4 shadow-sm">
    <h2 class="text-xl font-bold">Time Summary</h2>

    {#if loading}
      <div>Loading…</div>
    {:else}
      <!-- Active filters -->
      {#if selectedDivision || selectedTimeType || selectedName || selectedCategory}
        <div class="flex flex-wrap gap-2">
          {#if selectedDivision}
            <button on:click={() => clearFilter("division")} title="Clear division filter">
              <DsLabel color="blue" style="inverted">{selectedDivision.code} ✕</DsLabel>
            </button>
          {/if}
          {#if selectedTimeType}
            <button on:click={() => clearFilter("time_type")} title="Clear time type filter">
              <DsLabel color="green" style="inverted">{selectedTimeType.code} ✕</DsLabel>
            </button>
          {/if}
          {#if selectedName}
            <button on:click={() => clearFilter("name")} title="Clear name filter">
              <DsLabel color="purple" style="inverted">{selectedName.name} ✕</DsLabel>
            </button>
          {/if}
          {#if selectedCategory}
            <button on:click={() => clearFilter("category")} title="Clear category filter">
              <DsLabel color="red" style="inverted">{selectedCategory.name} ✕</DsLabel>
            </button>
          {/if}
        </div>
      {/if}

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

      <!-- Clickable lists -->
      {#if summary.divisions && summary.divisions.length > 0}
        <div class="flex flex-wrap items-center gap-2 pt-2">
          <span class="font-semibold">Divisions:</span>
          {#each summary.divisions as d}
            <button on:click={() => toggleFilter("division", d)} class="focus:outline-none">
              <DsLabel color="blue" style={selectedDivision?.id === d.id ? "inverted" : undefined}
                >{d.code}</DsLabel
              >
            </button>
          {/each}
        </div>
      {/if}

      {#if summary.time_types && summary.time_types.length > 0}
        <div class="flex flex-wrap items-center gap-2 pt-2">
          <span class="font-semibold">Time Types:</span>
          {#each summary.time_types as tt}
            <button on:click={() => toggleFilter("time_type", tt)} class="focus:outline-none">
              <DsLabel color="green" style={selectedTimeType?.id === tt.id ? "inverted" : undefined}
                >{tt.code}</DsLabel
              >
            </button>
          {/each}
        </div>
      {/if}

      {#if summary.names && summary.names.length > 0}
        <div class="flex flex-wrap items-center gap-2 pt-2">
          <span class="font-semibold">Staff:</span>
          {#each summary.names as n}
            <button on:click={() => toggleFilter("name", n)} class="focus:outline-none">
              <DsLabel color="purple" style={selectedName?.id === n.id ? "inverted" : undefined}
                >{n.name}</DsLabel
              >
            </button>
          {/each}
        </div>
      {/if}

      {#if summary.categories && summary.categories.length > 0}
        <div class="flex flex-wrap items-center gap-2 pt-2">
          <span class="font-semibold">Categories:</span>
          {#each summary.categories as c}
            <button on:click={() => toggleFilter("category", c)} class="focus:outline-none">
              <DsLabel color="red" style={selectedCategory?.id === c.id ? "inverted" : undefined}
                >{c.name}</DsLabel
              >
            </button>
          {/each}
        </div>
      {/if}
    {/if}
  </div>
</div>
