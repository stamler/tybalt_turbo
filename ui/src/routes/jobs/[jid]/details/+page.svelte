<script lang="ts">
  import type { PageData } from "./$types";
  import { onMount } from "svelte";
  import DSTabBar, { type TabItem } from "$lib/components/DSTabBar.svelte";
  import DSDateInput from "$lib/components/DSDateInput.svelte";
  import ClientNotesSection from "$lib/components/ClientNotesSection.svelte";
  import JobHeader from "$lib/components/jobs/JobHeader.svelte";
  import JobSummary from "$lib/components/jobs/JobSummary.svelte";
  import JobValueProposal from "$lib/components/jobs/JobValueProposal.svelte";
  import JobValueStatus from "$lib/components/jobs/JobValueStatus.svelte";
  import { JobValue, valueBlockers } from "$lib/jobValue.svelte";
  import JobDetailTab from "$lib/components/jobs/JobDetailTab.svelte";
  import TimeTabContent from "$lib/components/jobs/TimeTabContent.svelte";
  import ExpensesTabContent from "$lib/components/jobs/ExpensesTabContent.svelte";
  import POsTabContent from "$lib/components/jobs/POsTabContent.svelte";
  import WIPContent from "$lib/components/jobs/WIPContent.svelte";
  import StaffSummaryContent from "$lib/components/jobs/StaffSummaryContent.svelte";
  import DivisionsSummaryContent from "$lib/components/jobs/DivisionsSummaryContent.svelte";
  import { pb } from "$lib/pocketbase";
  import type { FilterDef } from "$lib/components/jobs/types";

  let { data }: { data: PageData } = $props();
  let isProposal = $derived(data.job.number?.startsWith("P") ?? false);
  // Value changes for this project, one instance per job so a late response
  // for the previous job cannot replace this one. The list only holds changes
  // the signed-in user may see.
  const jobValue = $derived(isProposal ? null : new JobValue(data.job.id));
  $effect(() => {
    jobValue?.reload();
  });
  type TabContentProps = {
    summary: Record<string, any>;
    items: any[];
    listLoading: boolean;
    loadMore: () => void;
    page: number;
    totalPages: number;
  };

  // Tab management ------------------------------------------------------------
  let activeTab = $state<"time" | "expenses" | "pos" | "wip">("time");
  let timeSubTab = $state<"all" | "staff_summary" | "divisions_summary">("all");

  // Reactive tabs array consumed by DSTabBar
  let tabs: TabItem[] = $derived([
    { label: "Time", href: "#time", active: activeTab === "time" },
    { label: "Expenses", href: "#expenses", active: activeTab === "expenses" },
    { label: "Active POs", href: "#pos", active: activeTab === "pos" },
    { label: "WIP", href: "#wip", active: activeTab === "wip" },
  ]);

  // Secondary tabs under Time
  let timeTabs: TabItem[] = $derived([
    { label: "All", href: "#time_all", active: timeSubTab === "all" },
    { label: "Staff summary", href: "#staff_summary", active: timeSubTab === "staff_summary" },
    {
      label: "Divisions summary",
      href: "#divisions_summary",
      active: timeSubTab === "divisions_summary",
    },
  ]);

  // Date range for summaries (persist between staff/divisions)
  let timeRangeStart = $state("");
  let timeRangeEnd = $state("");

  async function initTimeRange() {
    try {
      // add a no-op param to avoid PB SDK auto-cancelling the All tab's
      // identical summary request
      const res: any = await pb.send(`/api/jobs/${data.job.id}/time/summary?_init=1`, {
        method: "GET",
      });
      if (res?.earliest_entry) timeRangeStart = res.earliest_entry;
      if (res?.latest_entry) timeRangeEnd = res.latest_entry;
    } catch (err) {
      console.error("Failed to initialize time range", err);
    }
  }

  onMount(() => {
    // Initialize active tab based on hash
    if (typeof window !== "undefined") {
      const hash = window.location.hash;
      if (hash === "#expenses") {
        activeTab = "expenses";
      } else if (hash === "#pos") {
        activeTab = "pos";
      } else if (hash === "#wip") {
        activeTab = "wip";
      } else {
        activeTab = "time";
        if (hash === "#staff_summary") timeSubTab = "staff_summary";
        else if (hash === "#divisions_summary") timeSubTab = "divisions_summary";
        else if (hash === "#time_all" || hash === "#time" || hash === "") timeSubTab = "all";
        else timeSubTab = "all";
      }
    }

    // Listen for hash changes to update the active tab
    let handler: ((this: Window, ev: HashChangeEvent) => any) | null = null;
    if (typeof window !== "undefined") {
      handler = () => {
        const hash = window.location.hash;
        if (hash === "#expenses") {
          activeTab = "expenses";
        } else if (hash === "#pos") {
          activeTab = "pos";
        } else if (hash === "#wip") {
          activeTab = "wip";
        } else {
          activeTab = "time";
          if (hash === "#staff_summary") timeSubTab = "staff_summary";
          else if (hash === "#divisions_summary") timeSubTab = "divisions_summary";
          else if (hash === "#time_all" || hash === "#time" || hash === "") timeSubTab = "all";
          else timeSubTab = "all";
        }
      };
      window.addEventListener("hashchange", handler);
    }

    // Initialize default date range from time summary so summary subtabs
    // have values ready when first opened.
    initTimeRange();

    return () => {
      if (handler && typeof window !== "undefined") {
        window.removeEventListener("hashchange", handler);
      }
    };
  });

  // Lazily initialize the date range only when entering the summary subtabs
  // and only if the values are still empty.
  $effect(() => {
    if (
      activeTab === "time" &&
      (timeSubTab === "staff_summary" || timeSubTab === "divisions_summary")
    ) {
      if (!timeRangeStart || !timeRangeEnd) {
        initTimeRange();
      }
    }
  });

  // No-op: JobDetailTab initializes itself on first activation. Avoid forcing
  // repeated refreshes that could trigger PB auto-cancel cascades.

  // Filter Definitions --------------------------------------------------------
  const divisionFilter: FilterDef = {
    type: "division",
    label: "Divisions",
    summaryProperty: "divisions",
    valueProperty: "id",
    displayProperty: "code",
    color: "blue",
  };

  const staffFilter: FilterDef = {
    type: "name",
    label: "Staff",
    queryParam: "uid",
    summaryProperty: "names",
    valueProperty: "id",
    displayProperty: "name",
    color: "purple",
  };

  const categoryFilter: FilterDef = {
    type: "category",
    label: "Categories",
    summaryProperty: "categories",
    valueProperty: "id",
    displayProperty: "name",
    color: "teal",
  };

  const branchFilter: FilterDef = {
    type: "branch",
    label: "Branches",
    summaryProperty: "branches",
    valueProperty: "id",
    displayProperty: "code",
    color: "gray",
  };

  const timeFilterDefs: FilterDef[] = [
    divisionFilter,
    {
      type: "time_type",
      label: "Time Types",
      summaryProperty: "time_types",
      valueProperty: "id",
      displayProperty: "code",
      color: "green",
    },
    staffFilter,
    categoryFilter,
    branchFilter,
  ];

  const expenseFilterDefs: FilterDef[] = [
    divisionFilter,
    {
      type: "payment_type",
      label: "Payment Types",
      summaryProperty: "payment_types",
      valueProperty: "name",
      displayProperty: "name",
      color: "green",
    },
    staffFilter,
    categoryFilter,
    branchFilter,
  ];

  const poFilterDefs: FilterDef[] = [
    divisionFilter,
    {
      type: "type",
      label: "Types",
      summaryProperty: "types",
      valueProperty: "name",
      displayProperty: "name",
      color: "green",
    },
    staffFilter,
    branchFilter,
  ];
</script>

<div class="mx-auto space-y-4 p-4">
  {#key data.job.id}
    <JobHeader job={data.job} />
    {#if jobValue}<JobValueStatus value={jobValue} />{/if}
    <JobSummary job={data.job} {jobValue} />
  {/key}

  <ClientNotesSection
    clientId={data.job.client?.id ?? ""}
    notes={data.notes}
    jobOptions={[]}
    preselectedJobId={data.job.id}
    heading="Notes"
    notesEndpoint={`/api/jobs/${data.job.id}/notes`}
  />

  {#if jobValue}
    {#key data.job.id}
      <JobValueProposal
        value={jobValue}
        effectiveValue={data.job.project_value ?? 0}
        missing={valueBlockers(data.job)}
      />
    {/key}
  {/if}

  <!-- Tab Bar -->
  <DSTabBar {tabs} />

  <!-- Time Section -->
  <div id="time" class:hidden={activeTab !== "time"}>
    <!-- Secondary Tab Bar under Time -->
    <div class="mt-2">
      <DSTabBar tabs={timeTabs} />
    </div>

    <!-- All (existing content) -->
    <div class:hidden={timeSubTab !== "all"}>
      {#key data.job.id}
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
            <TimeTabContent
              {summary}
              {items}
              {listLoading}
              {loadMore}
              {page}
              {totalPages}
              jobId={data.job.id}
            />
          {/snippet}
        </JobDetailTab>
      {/key}
    </div>

    <!-- Staff summary -->
    <div id="staff_summary" class:hidden={timeSubTab !== "staff_summary"}>
      <div class="flex flex-wrap items-end gap-3 px-4 py-2">
        <div>
          <label for="staff-start-date" class="block text-sm font-semibold">Start date</label>
          <DSDateInput
            id="staff-start-date"
            bind:value={timeRangeStart}
            class="rounded-sm border px-2 py-1"
          />
        </div>
        <div>
          <label for="staff-end-date" class="block text-sm font-semibold">End date</label>
          <DSDateInput
            id="staff-end-date"
            bind:value={timeRangeEnd}
            class="rounded-sm border px-2 py-1"
          />
        </div>
      </div>
      <StaffSummaryContent
        rateSheet={data.job.rate_sheet}
        jobId={data.job.id}
        startDate={timeRangeStart}
        endDate={timeRangeEnd}
      />
    </div>

    <!-- Divisions summary -->
    <div id="divisions_summary" class:hidden={timeSubTab !== "divisions_summary"}>
      <div class="flex flex-wrap items-end gap-3 px-4 py-2">
        <div>
          <label for="div-start-date" class="block text-sm font-semibold">Start date</label>
          <DSDateInput
            id="div-start-date"
            bind:value={timeRangeStart}
            class="rounded-sm border px-2 py-1"
          />
        </div>
        <div>
          <label for="div-end-date" class="block text-sm font-semibold">End date</label>
          <DSDateInput
            id="div-end-date"
            bind:value={timeRangeEnd}
            class="rounded-sm border px-2 py-1"
          />
        </div>
      </div>
      <DivisionsSummaryContent
        rateSheet={data.job.rate_sheet}
        jobId={data.job.id}
        startDate={timeRangeStart}
        endDate={timeRangeEnd}
      />
    </div>
  </div>

  <!-- Expenses Section -->
  <div id="expenses" class:hidden={activeTab !== "expenses"}>
    {#key data.job.id}
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
    {/key}
  </div>

  <!-- POs Section -->
  <div id="pos" class:hidden={activeTab !== "pos"}>
    {#key data.job.id}
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
    {/key}
  </div>
  {#if activeTab === "wip"}
    <div id="wip">
      {#key data.job}
        <WIPContent jobId={data.job.id} />
      {/key}
    </div>
  {/if}
  <!-- Jobs list section -->
</div>
