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
  import StaffSummaryContent from "$lib/components/jobs/StaffSummaryContent.svelte";
  import DivisionsSummaryContent from "$lib/components/jobs/DivisionsSummaryContent.svelte";
  import DSLocationPicker from "$lib/components/DSLocationPicker.svelte";
  import { pb } from "$lib/pocketbase";
  import { goto, invalidateAll } from "$app/navigation";
  import type { FilterDef } from "$lib/components/jobs/types";
  import { formatCurrency, shortDate } from "$lib/utilities";
  import ClientNotesSection from "$lib/components/ClientNotesSection.svelte";
  import { JobsStatusOptions } from "$lib/pocketbase-types";
  let awarding = $state(false);
  let validatingForProject = $state(false);

  // Validate proposal and redirect to create project or edit page
  async function handleCreateReferencingProject() {
    if (validatingForProject) return;
    validatingForProject = true;
    try {
      const response = await pb.send(`/api/jobs/${data.job.id}/validate-proposal`, {
        method: "GET",
      });
      if (response.valid) {
        // Proposal is valid - redirect to create project with today's award date
        await goto(`/jobs/add/from/${data.job.id}?setAwardToday=true`);
      } else {
        // Proposal has validation errors - store errors and redirect flag, then go to edit page
        if (typeof sessionStorage !== "undefined") {
          if (response.errors) {
            sessionStorage.setItem(
              `proposal_validation_errors_${data.job.id}`,
              JSON.stringify(response.errors),
            );
          }
          // Flag to redirect back to create project page after successful save
          sessionStorage.setItem(`redirect_to_create_project_${data.job.id}`, "true");
        }
        await goto(`/jobs/${data.job.id}/edit`);
      }
    } catch (e) {
      console.error("Failed to validate proposal", e);
      // On error, redirect to edit page as a fallback
      await goto(`/jobs/${data.job.id}/edit`);
    } finally {
      validatingForProject = false;
    }
  }

  let { data }: { data: PageData } = $props();
  let isProposal = $derived(data.job.number?.startsWith("P") ?? false);

  type TabContentProps = {
    summary: Record<string, any>;
    items: any[];
    listLoading: boolean;
    loadMore: () => void;
    page: number;
    totalPages: number;
  };

  function personName(person: any) {
    if (!person) return "";
    return `${person.given_name || person.name || ""} ${person.surname || ""}`.trim();
  }

  // Tab management ------------------------------------------------------------
  let activeTab = $state<"time" | "expenses" | "pos">("time");
  let timeSubTab = $state<"all" | "staff_summary" | "divisions_summary">("all");

  // Reactive tabs array consumed by DSTabBar
  let tabs: TabItem[] = $derived([
    { label: "Time", href: "#time", active: activeTab === "time" },
    { label: "Expenses", href: "#expenses", active: activeTab === "expenses" },
    { label: "Active POs", href: "#pos", active: activeTab === "pos" },
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
  <div class="flex items-center justify-between gap-2">
    <h1 class="text-2xl font-bold">Job Details</h1>
    <div class="flex items-center gap-2">
      {#if data.job.number?.startsWith("P")}
        <DsActionButton
          action={handleCreateReferencingProject}
          icon="mdi:briefcase-plus"
          title="Create referencing project"
          color="yellow"
          loading={validatingForProject}
        />
      {/if}
      {#if data.job.number?.startsWith("P") && data.job.status === "Awarded"}
        <DsActionButton
          action={`/jobs/add/from/${data.job.id}`}
          icon="mdi:plus"
          title="Create Project from Proposal"
          color="green"
        />
      {:else if data.job.number?.startsWith("P") && data.job.status === "Submitted"}
        {#key data.job.id}
          <DsActionButton
            action={async () => {
              try {
                awarding = true;
                await pb
                  .collection("jobs")
                  .update(data.job.id, { status: JobsStatusOptions.Awarded });
                // Re-run this page's load so `/api/jobs/{id}/details` fetches fresh data and the
                // "Create Project from Proposal" button appears without a full page reload.
                // Prefer invalidateAll() over window.location.reload() to preserve UX where possible.
                await invalidateAll();
              } catch (e) {
                console.error("Failed to award proposal", e);
                typeof window !== "undefined" &&
                  window.alert("Failed to award proposal. Please try again.");
              } finally {
                awarding = false;
              }
            }}
            icon="mdi:trophy"
            title="Award proposal"
            color="yellow"
            loading={awarding}
          />
        {/key}
      {/if}
      <DsActionButton
        action={`/jobs/${data.job.id}/edit`}
        icon="mdi:pencil"
        title="Edit Job"
        color="blue"
      />
    </div>
  </div>

  <div class="space-y-2 rounded-sm bg-neutral-100 p-4">
    <div class="flex items-center gap-2">
      <span class="font-semibold">Job Number:</span>
      <span>{data.job.number}</span>
      {#if data.job.number?.startsWith("P")}
        <DsLabel color="yellow">proposal</DsLabel>
      {/if}
      {#if data.job.number && !data.job.number.startsWith("P") && /^\d{2}-\d{3,4}$/.test(data.job.number)}
        <a href={`/jobs/add/${data.job.id}`} class="ml-2 text-blue-600 underline">Create Sub-Job</a>
      {/if}
    </div>
    <div><span class="font-semibold">Description:</span> {data.job.description}</div>
    {#if data.job.status}
      <div><span class="font-semibold">Status:</span> {data.job.status}</div>
    {/if}

    <details class="space-y-2">
      <summary class="cursor-pointer font-semibold text-neutral-700">Additional Details</summary>
      <div class="space-y-2">
        {#if data.job.branch_name || data.job.branch_code || data.job.branch_id}
          <div>
            <span class="font-semibold">Branch:</span>
            {#if data.job.branch_name}
              {data.job.branch_name}
              {#if data.job.branch_code}
                ({data.job.branch_code})
              {/if}
            {:else if data.job.branch_code}
              {data.job.branch_code}
            {:else}
              {data.job.branch_id}
            {/if}
          </div>
        {/if}

        {#if data.job.client}
          <div>
            <span class="font-semibold">Client:</span>
            <a
              href={`/clients/${data.job.client.id}/details`}
              class="text-blue-600 hover:underline"
            >
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

        {#if data.job.job_owner && data.job.job_owner.id}
          <div>
            <span class="font-semibold">Job Owner:</span>
            <a
              href={`/clients/${data.job.job_owner.id}/details`}
              class="text-blue-600 hover:underline"
            >
              {data.job.job_owner.name}
            </a>
          </div>
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

        {#if !isProposal}
          {#if data.job.authorizing_document}
            <div>
              <span class="font-semibold">Authorizing Document:</span>
              {data.job.authorizing_document}
            </div>
          {/if}
          {#if data.job.authorizing_document === "PO" && data.job.client_po}
            <div>
              <span class="font-semibold">Client PO:</span>
              {data.job.client_po}
            </div>
          {/if}
          {#if data.job.client_reference_number}
            <div>
              <span class="font-semibold">Client Reference Number:</span>
              {data.job.client_reference_number}
            </div>
          {/if}

          <div>
            <span class="font-semibold">Outstanding Balance:</span>
            {formatCurrency(data.job.outstanding_balance ?? 0)}
            {#if data.job.outstanding_balance_date}
              <span class="text-sm text-neutral-500">
                (As of {shortDate(data.job.outstanding_balance_date, true)})
              </span>
            {/if}
          </div>

          {#if data.job.rate_sheet?.id}
            <div>
              <span class="font-semibold">Rate Sheet:</span>
              <a
                href={`/rate-sheets/${data.job.rate_sheet.id}/details`}
                class="text-blue-600 hover:underline"
              >
                {data.job.rate_sheet.name} (rev. {data.job.rate_sheet.revision})
              </a>
            </div>
          {/if}
        {/if}

        {#if data.job.categories && data.job.categories.length > 0}
          <div class="flex items-start gap-2">
            <span class="pt-1 font-semibold">Categories:</span>
            <div class="flex flex-wrap gap-1">
              {#each data.job.categories as category}
                <DsLabel color="blue">{category.name}</DsLabel>
              {/each}
            </div>
          </div>
        {/if}

        {#if data.job.children && data.job.children.length > 0}
          <div>
            <span class="font-semibold">Children:</span>
            <span>
              {#each data.job.children as c, i}
                <a href={`/jobs/${c.id}/details`} class="text-blue-600 hover:underline"
                  >{c.number}</a
                >{i < data.job.children.length - 1 ? ", " : ""}
              {/each}
            </span>
          </div>
        {/if}

        {#if data.job.parent_id}
          <div>
            <span class="font-semibold">Parent Job:</span>
            <a href={`/jobs/${data.job.parent_id}/details`} class="text-blue-600 hover:underline">
              {data.job.parent_number || data.job.parent_id}
            </a>
          </div>
        {/if}

        {#if !isProposal && data.job.project_award_date}
          <div>
            <span class="font-semibold">Project Award Date:</span>
            {data.job.project_award_date}
          </div>
        {/if}

        {#if isProposal}
          <div>
            <span class="font-semibold">Proposal Value:</span>
            {formatCurrency(data.job.proposal_value ?? 0)}
          </div>
          <div>
            <span class="font-semibold">Time & Materials:</span>
            {data.job.time_and_materials ? "Yes" : "No"}
          </div>
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
        {/if}

        {#if data.job.allocations && Array.isArray(data.job.allocations) && data.job.allocations.length > 0}
          <div>
            <span class="font-semibold">Divisions:</span>
            <div class="mt-1 flex flex-col gap-1">
              {#each data.job.allocations as a}
                <div class="flex items-center gap-2">
                  <span>{a.division?.name} ({a.division?.code})</span>
                  <span class="text-neutral-600">— {a.hours} h</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        {#if data.job.projects && data.job.projects.length > 0}
          <div>
            <span class="font-semibold">Projects:</span>
            {#each data.job.projects as p, i}
              <a href={`/jobs/${p.id}/details`} class="text-blue-600 hover:underline">{p.number}</a
              >{i < data.job.projects.length - 1 ? ", " : ""}
            {/each}
          </div>
        {/if}

        {#if data.job.location && data.job.location !== ""}
          <div class="mt-2">
            <span class="font-semibold">Location:</span>
            <div class="mt-1">
              <DSLocationPicker
                value={data.job.location}
                errors={{}}
                fieldName="location"
                disabled={true}
                readonly={true}
              />
            </div>
          </div>
        {/if}
      </div>
    </details>

    <ClientNotesSection
      clientId={data.job.client?.id ?? ""}
      notes={data.notes}
      jobOptions={[]}
      preselectedJobId={data.job.id}
      heading="Notes"
      notesEndpoint={`/api/jobs/${data.job.id}/notes`}
    />
  </div>

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
          <input
            id="staff-start-date"
            type="date"
            bind:value={timeRangeStart}
            class="rounded-sm border px-2 py-1"
          />
        </div>
        <div>
          <label for="staff-end-date" class="block text-sm font-semibold">End date</label>
          <input
            id="staff-end-date"
            type="date"
            bind:value={timeRangeEnd}
            class="rounded-sm border px-2 py-1"
          />
        </div>
      </div>
      <StaffSummaryContent jobId={data.job.id} startDate={timeRangeStart} endDate={timeRangeEnd} />
    </div>

    <!-- Divisions summary -->
    <div id="divisions_summary" class:hidden={timeSubTab !== "divisions_summary"}>
      <div class="flex flex-wrap items-end gap-3 px-4 py-2">
        <div>
          <label for="div-start-date" class="block text-sm font-semibold">Start date</label>
          <input
            id="div-start-date"
            type="date"
            bind:value={timeRangeStart}
            class="rounded-sm border px-2 py-1"
          />
        </div>
        <div>
          <label for="div-end-date" class="block text-sm font-semibold">End date</label>
          <input
            id="div-end-date"
            type="date"
            bind:value={timeRangeEnd}
            class="rounded-sm border px-2 py-1"
          />
        </div>
      </div>
      <DivisionsSummaryContent
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
  <!-- Jobs list section -->
</div>
