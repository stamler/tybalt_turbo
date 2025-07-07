<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import type { PageData } from "./$types";
  import { onMount } from "svelte";
  import DSTabBar, { type TabItem } from "$lib/components/DSTabBar.svelte";
  import JobDetailTab from "$lib/components/jobs/JobDetailTab.svelte";
  import TimeTabContent from "$lib/components/jobs/TimeTabContent.svelte";
  import ExpensesTabContent from "$lib/components/jobs/ExpensesTabContent.svelte";
  import POsTabContent from "$lib/components/jobs/POsTabContent.svelte";
  import type { FilterDef } from "$lib/components/jobs/types";

  type TabContentProps = {
    summary: Record<string, any>;
    items: any[];
    listLoading: boolean;
    loadMore: () => void;
    page: number;
    totalPages: number;
  };

  export let data: PageData;

  function personName(person: any) {
    if (!person) return "";
    return `${person.given_name || person.name || ""} ${person.surname || ""}`.trim();
  }

  // Tab management ------------------------------------------------------------
  let activeTab: "time" | "expenses" | "pos" = "time";

  // Reactive tabs array consumed by DSTabBar
  let tabs: TabItem[] = [];
  $: tabs = [
    { label: "Time", href: "#time", active: activeTab === "time" },
    { label: "Expenses", href: "#expenses", active: activeTab === "expenses" },
    { label: "Purchase Orders", href: "#pos", active: activeTab === "pos" },
  ];

  onMount(() => {
    // Initialize active tab based on hash
    if (typeof window !== "undefined") {
      const hash = window.location.hash;
      if (hash === "#expenses") activeTab = "expenses";
      else if (hash === "#pos") activeTab = "pos";
      else activeTab = "time";
    }

    // Listen for hash changes to update the active tab
    if (typeof window !== "undefined") {
      window.addEventListener("hashchange", () => {
        const hash = window.location.hash;
        if (hash === "#expenses") activeTab = "expenses";
        else if (hash === "#pos") activeTab = "pos";
        else activeTab = "time";
      });
    }
  });

  // Filter Definitions --------------------------------------------------------
  const timeFilterDefs: FilterDef[] = [
    {
      type: "division",
      label: "Divisions",
      summaryProperty: "divisions",
      valueProperty: "id",
      displayProperty: "code",
      color: "blue",
    },
    {
      type: "time_type",
      label: "Time Types",
      summaryProperty: "time_types",
      valueProperty: "id",
      displayProperty: "code",
      color: "green",
    },
    {
      type: "name",
      label: "Staff",
      queryParam: "uid",
      summaryProperty: "names",
      valueProperty: "id",
      displayProperty: "name",
      color: "purple",
    },
    {
      type: "category",
      label: "Categories",
      summaryProperty: "categories",
      valueProperty: "id",
      displayProperty: "name",
      color: "teal",
    },
  ];

  const expenseFilterDefs: FilterDef[] = [
    {
      type: "division",
      label: "Divisions",
      summaryProperty: "divisions",
      valueProperty: "id",
      displayProperty: "code",
      color: "blue",
    },
    {
      type: "payment_type",
      label: "Payment Types",
      summaryProperty: "payment_types",
      valueProperty: "name",
      displayProperty: "name",
      color: "green",
    },
    {
      type: "name",
      label: "Staff",
      queryParam: "uid",
      summaryProperty: "names",
      valueProperty: "id",
      displayProperty: "name",
      color: "purple",
    },
    {
      type: "category",
      label: "Categories",
      summaryProperty: "categories",
      valueProperty: "id",
      displayProperty: "name",
      color: "teal",
    },
  ];

  const poFilterDefs: FilterDef[] = [
    {
      type: "division",
      label: "Divisions",
      summaryProperty: "divisions",
      valueProperty: "id",
      displayProperty: "code",
      color: "blue",
    },
    {
      type: "type",
      label: "Types",
      summaryProperty: "types",
      valueProperty: "name",
      displayProperty: "name",
      color: "green",
    },
    {
      type: "name",
      label: "Staff",
      queryParam: "uid",
      summaryProperty: "names",
      valueProperty: "id",
      displayProperty: "name",
      color: "purple",
    },
  ];
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

  <!-- Tab Bar -->
  <DSTabBar {tabs} />

  <!-- Time Section -->
  <div id="time" class:hidden={activeTab !== "time"}>
    <JobDetailTab
      active={activeTab === "time"}
      jobId={data.job.id}
      summaryUrl={`/api/jobs/${data.job.id}/time/summary`}
      listUrl={`/api/jobs/${data.job.id}/time/entries`}
      filterDefs={timeFilterDefs}
    >
      {#snippet children({
        summary,
        items,
        listLoading,
        loadMore,
        page,
        totalPages,
      }: TabContentProps)}
        <TimeTabContent {summary} {items} {listLoading} {loadMore} {page} {totalPages} />
      {/snippet}
    </JobDetailTab>
  </div>

  <!-- Expenses Section -->
  <div id="expenses" class:hidden={activeTab !== "expenses"}>
    <JobDetailTab
      active={activeTab === "expenses"}
      jobId={data.job.id}
      summaryUrl={`/api/jobs/${data.job.id}/expenses/summary`}
      listUrl={`/api/jobs/${data.job.id}/expenses/list`}
      filterDefs={expenseFilterDefs}
    >
      {#snippet children({
        summary,
        items,
        listLoading,
        loadMore,
        page,
        totalPages,
      }: TabContentProps)}
        <ExpensesTabContent {summary} {items} {listLoading} {loadMore} {page} {totalPages} />
      {/snippet}
    </JobDetailTab>
  </div>

  <!-- POs Section -->
  <div id="pos" class:hidden={activeTab !== "pos"}>
    <JobDetailTab
      active={activeTab === "pos"}
      jobId={data.job.id}
      summaryUrl={`/api/jobs/${data.job.id}/pos/summary`}
      listUrl={`/api/jobs/${data.job.id}/pos/list`}
      filterDefs={poFilterDefs}
    >
      {#snippet children({
        summary,
        items,
        listLoading,
        loadMore,
        page,
        totalPages,
      }: TabContentProps)}
        <POsTabContent {summary} {items} {listLoading} {loadMore} {page} {totalPages} />
      {/snippet}
    </JobDetailTab>
  </div>
</div>
