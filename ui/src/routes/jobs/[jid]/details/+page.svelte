<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DSPopover from "$lib/components/DSPopover.svelte";
  import DSTextInput from "$lib/components/DSTextInput.svelte";
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
  import DSDateInput from "$lib/components/DSDateInput.svelte";
  import FastCloseConfirmPopover from "$lib/components/FastCloseConfirmPopover.svelte";
  import StoredFileHashRepairPopover from "$lib/components/StoredFileHashRepairPopover.svelte";
  import { pb } from "$lib/pocketbase";
  import { goto, invalidateAll } from "$app/navigation";
  import { globalStore } from "$lib/stores/global";
  import { jobsEditingEnabled } from "$lib/stores/appConfig";
  import type { FilterDef } from "$lib/components/jobs/types";
  import { formatCurrency, shortDate } from "$lib/utilities";
  import ClientNotesSection from "$lib/components/ClientNotesSection.svelte";
  import { JobsStatusOptions } from "$lib/pocketbase-types";
  import Icon from "@iconify/svelte";
  let awarding = $state(false);
  let validatingForProject = $state(false);
  let closingImportedProject = $state(false);
  let showFastCloseConfirm = $state(false);
  let fastCloseContextLoading = $state(false);
  let fastCloseContextError = $state<string | null>(null);
  let showSetNumberModal = $state(false);
  let setNumberSubmitting = $state(false);
  let setNumberValue = $state("");
  let setNumberGlobalError = $state<string | null>(null);
  let setNumberErrors = $state({} as Record<string, { message: string }>);
  let paUploading = $state(false);
  let paUploadError = $state<string | null>(null);
  let paRevoking = $state(false);
  let showPARevokeConfirm = $state(false);
  let paRevokeError = $state<string | null>(null);
  let paDeleting = $state(false);
  let showPADeleteConfirm = $state(false);
  let paDeleteError = $state<string | null>(null);
  let showPAHashRepairPopover = $state(false);
  let fastCloseProposal = $state<{
    id: string;
    number: string;
    status: string;
    imported: boolean;
  } | null>(null);

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
  const canFastCloseImportedProject = $derived(
    !isProposal && data.job.status === "Active" && data.job.imported === true,
  );
  const canSetNumber = $derived($jobsEditingEnabled && $globalStore.claims.includes("admin"));
  const currentUserID = $derived(pb.authStore.record?.id ?? "");
  const canUploadProjectAuthorization = $derived(
    !isProposal &&
      (Boolean(currentUserID) &&
        ($globalStore.claims.includes("job") ||
          currentUserID === data.job.manager?.id ||
          currentUserID === data.job.alternate_manager?.id ||
          currentUserID === data.job.branch_manager_id)),
  );
  const projectAuthorizationApproved = $derived(
    Boolean(
      data.job.project_authorization_doc &&
        data.job.project_authorization_doc_hash &&
        data.job.pa_reviewed &&
        data.job.pa_reviewer?.id,
    ),
  );
  const canRevokeProjectAuthorization = $derived(
    !isProposal && $globalStore.claims.includes("admin") && projectAuthorizationApproved,
  );
  const canRepairProjectAuthorizationHash = $derived(
    !isProposal &&
      $globalStore.claims.includes("admin") &&
      Boolean(data.job.project_authorization_doc),
  );
  const canDeleteProjectAuthorization = $derived(
    canUploadProjectAuthorization &&
      Boolean(data.job.project_authorization_doc) &&
      !projectAuthorizationApproved,
  );
  const hasNumberHierarchyWarning = $derived(
    Boolean(data.job.parent_number) ||
      (Array.isArray(data.job.children) && data.job.children.length > 0),
  );

  // Load proposal context and open confirmation modal so users can see exact
  // side effects before executing a potentially destructive close.
  async function openFastCloseConfirm() {
    showFastCloseConfirm = true;
    fastCloseContextLoading = true;
    fastCloseContextError = null;
    fastCloseProposal = null;

    try {
      const proposalId = data.job.proposal_id;
      if (proposalId) {
        const proposal = await pb.send(`/api/jobs/${proposalId}`, { method: "GET" });
        fastCloseProposal = {
          id: proposal.id,
          number: data.job.proposal_number || proposal.number || proposalId,
          status: proposal.status,
          imported: Boolean(proposal.imported),
        };
      }
    } catch (error: any) {
      fastCloseContextError =
        error?.response?.message ??
        error?.response?.error ??
        error?.message ??
        "Unable to load referenced proposal details.";
    } finally {
      fastCloseContextLoading = false;
    }
  }

  function closeFastCloseConfirm() {
    showFastCloseConfirm = false;
    fastCloseContextLoading = false;
    fastCloseContextError = null;
    fastCloseProposal = null;
  }

  function openSetNumberModal() {
    setNumberValue = data.job.number ?? "";
    setNumberGlobalError = null;
    setNumberErrors = {};
    showSetNumberModal = true;
  }

  function closeSetNumberModal() {
    if (setNumberSubmitting) return;
    showSetNumberModal = false;
    setNumberGlobalError = null;
    setNumberErrors = {};
  }

  async function submitSetNumber() {
    if (setNumberSubmitting) return;

    setNumberSubmitting = true;
    setNumberGlobalError = null;
    setNumberErrors = {};

    try {
      await pb.send(`/api/jobs/${data.job.id}/set-number`, {
        method: "POST",
        body: { number: setNumberValue },
      });
      closeSetNumberModal();
      await invalidateAll();
    } catch (error: any) {
      const backendErrors = error?.data?.data as Record<string, { message: string }> | undefined;
      const backendMessage = error?.data?.message ?? error?.data?.error ?? error?.message;
      setNumberErrors = backendErrors ?? {};
      if (!backendErrors || Object.keys(backendErrors).length === 0) {
        setNumberGlobalError = backendMessage ?? "Failed to change job number";
      }
    } finally {
      setNumberSubmitting = false;
    }
  }

  // Close an imported Active project through the dedicated backend flow.
  //
  // Why a dedicated handler:
  // - The project detail page should expose the same "legacy-only fast close"
  //   policy as the jobs list.
  // - We intentionally call /api/jobs/{id}/close instead of patching the job
  //   record directly to guarantee the transactional behavior (close + notes +
  //   optional proposal auto-award) defined by backend policy.
  async function handleFastCloseConfirmSubmit() {
    if (closingImportedProject || fastCloseContextLoading) return;
    closingImportedProject = true;
    try {
      await pb.send(`/api/jobs/${data.job.id}/close`, { method: "POST" });
      await invalidateAll();
      closeFastCloseConfirm();
    } catch (error: any) {
      const msg =
        error?.response?.message ??
        error?.response?.error ??
        error?.message ??
        "Failed to close imported project";
      globalStore.addError(msg);
    } finally {
      closingImportedProject = false;
    }
  }

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

  function projectAuthorizationStatus() {
    if (!data.job.project_authorization_doc) return "PA document missing";
    if (projectAuthorizationApproved) return "PA approved";
    return "PA pending Accounting approval";
  }

  async function uploadProjectAuthorizationDoc(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    paUploading = true;
    paUploadError = null;
    try {
      const form = new FormData();
      form.append("project_authorization_doc", file);
      await pb.send(`/api/jobs/${data.job.id}/project_authorization_doc`, {
        method: "POST",
        body: form,
      });
      input.value = "";
      await globalStore.refreshAttentionCounts();
      await invalidateAll();
    } catch (error: any) {
      paUploadError =
        error?.data?.data?.project_authorization_doc?.message ??
        error?.data?.message ??
        error?.message ??
        "Failed to upload PA document.";
    } finally {
      paUploading = false;
    }
  }

  async function revokeProjectAuthorization() {
    paRevoking = true;
    paRevokeError = null;
    try {
      await pb.send(`/api/jobs/${data.job.id}/project_authorization/revoke`, { method: "POST" });
      showPARevokeConfirm = false;
      await globalStore.refreshAttentionCounts();
      await invalidateAll();
    } catch (error: any) {
      paRevokeError = error?.data?.message ?? error?.message ?? "Failed to revoke PA approval.";
    } finally {
      paRevoking = false;
    }
  }

  async function deleteProjectAuthorizationDoc() {
    paDeleting = true;
    paDeleteError = null;
    try {
      await pb.send(`/api/jobs/${data.job.id}/project_authorization_doc`, { method: "DELETE" });
      showPADeleteConfirm = false;
      await globalStore.refreshAttentionCounts();
      await invalidateAll();
    } catch (error: any) {
      paDeleteError =
        error?.data?.data?.project_authorization_doc?.message ??
        error?.data?.message ??
        error?.message ??
        "Failed to remove PA document.";
    } finally {
      paDeleting = false;
    }
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
      {#if canFastCloseImportedProject}
        <DsActionButton
          action={openFastCloseConfirm}
          icon="mdi:archive-check"
          title="Close imported project"
          color="red"
          loading={closingImportedProject}
        />
      {/if}
      <DsActionButton
        action={`/jobs/${data.job.id}/edit`}
        icon="mdi:pencil"
        title="Edit Job"
        color="blue"
      />
    </div>
  </div>

  <FastCloseConfirmPopover
    bind:show={showFastCloseConfirm}
    jobNumber={data.job.number ?? "Project"}
    proposal={fastCloseProposal}
    loadingContext={fastCloseContextLoading}
    contextError={fastCloseContextError}
    submitting={closingImportedProject}
    onSubmit={handleFastCloseConfirmSubmit}
    onCancel={closeFastCloseConfirm}
  />

  <DSPopover
    bind:show={showPARevokeConfirm}
    title="Revoke PA Approval"
    subtitle="Revoking approval may block new time bundles, purchase orders, and expenses for this project when PA enforcement is enabled."
    error={paRevokeError}
    submitting={paRevoking}
    submitLabel="Revoke Approval"
    onSubmit={revokeProjectAuthorization}
  >
    <p class="text-sm text-neutral-700">
      The uploaded PDF and hash will stay on the job. Accounting will need to approve the current
      document again before the project is treated as PA-approved.
    </p>
  </DSPopover>

  <DSPopover
    bind:show={showPADeleteConfirm}
    title="Remove PA PDF"
    subtitle="Removing the uploaded PA will leave this project without a pending document for Accounting review."
    error={paDeleteError}
    submitting={paDeleting}
    submitLabel="Remove PDF"
    onSubmit={deleteProjectAuthorizationDoc}
  >
    <p class="text-sm text-neutral-700">
      A new signed PA PDF will need to be uploaded before Accounting can approve the project.
    </p>
  </DSPopover>

  <StoredFileHashRepairPopover
    show={showPAHashRepairPopover}
    recordId={data.job.id}
    title="Project Authorization Document Repair"
    hasAttachment={Boolean(data.job.project_authorization_doc)}
    currentHash={data.job.project_authorization_doc_hash}
    currentUpdated={data.job.updated}
    auditPath={`/api/jobs/${data.job.id}/project_authorization_doc_hash/audit`}
    replacePath={`/api/jobs/${data.job.id}/project_authorization_doc_hash/replace`}
    canMarkMissing={false}
    auditMatchesMessage="Stored hash matches the PA document."
    auditMismatchMessage="Stored hash does not match the PA document."
    replaceConfirmMessage="Replacing this stored hash is irreversible. Verify that the PA PDF opens and is actually usable before accepting the calculated hash."
    replaceNoopMessage="No change made. The stored hash already matches the PA document."
    replaceSuccessMessage="Stored hash replaced with the calculated PA document hash."
    onClose={() => (showPAHashRepairPopover = false)}
    onRepaired={invalidateAll}
  />

  <DSPopover
    bind:show={showSetNumberModal}
    title="Change Job Number"
    subtitle="Admins can manually override the stored job number without reopening the full editor."
    error={setNumberGlobalError}
    submitting={setNumberSubmitting}
    submitLabel="Save Number"
    onSubmit={submitSetNumber}
    onCancel={closeSetNumberModal}
  >
    <DSTextInput
      bind:value={setNumberValue}
      errors={setNumberErrors}
      fieldName="number"
      uiName="Job Number"
      disabled={setNumberSubmitting}
    />

    {#if hasNumberHierarchyWarning}
      <div class="rounded-sm border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
        <p class="font-semibold">Related records are not renumbered automatically.</p>
        {#if data.job.parent_number}
          <p class="mt-1">This job has parent number {data.job.parent_number}.</p>
        {/if}
        {#if Array.isArray(data.job.children) && data.job.children.length > 0}
          <p class="mt-1">
            This job has {data.job.children.length} child job{data.job.children.length === 1
              ? ""
              : "s"}.
          </p>
        {/if}
        <p class="mt-1">If related job numbers should stay aligned, update those records separately.</p>
      </div>
    {/if}
  </DSPopover>

  <div class="space-y-2 rounded-sm bg-neutral-100 p-4">
    <div class="flex items-center gap-2">
      <span class="font-semibold">Job Number:</span>
      <span>{data.job.number}</span>
      {#if data.job.number?.startsWith("P")}
        <DsLabel color="yellow">proposal</DsLabel>
      {/if}
      {#if canSetNumber}
        <DsActionButton action={openSetNumberModal} color="yellow">Change Number</DsActionButton>
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
          {#if data.job.client_po}
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
          <div class="flex flex-col gap-2 rounded-sm border border-neutral-200 p-3">
            <div>
              <span class="font-semibold">PA Review:</span>
              {projectAuthorizationStatus()}
            </div>
            {#if data.job.project_authorization_doc_url}
              <a
                href={data.job.project_authorization_doc_url}
                target="_blank"
                rel="noreferrer"
                class="text-blue-600 hover:underline"
              >
                Open PA PDF
              </a>
            {/if}
            {#if data.job.project_authorization_doc_hash || canRepairProjectAuthorizationHash}
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-semibold">PA Hash:</span>
                {#if canRepairProjectAuthorizationHash}
                  <button
                    type="button"
                    class="font-mono text-sm text-blue-700 underline decoration-dotted underline-offset-2 hover:text-blue-900"
                    title="Audit PA document hash"
                    onclick={() => (showPAHashRepairPopover = true)}
                  >
                    {data.job.project_authorization_doc_hash
                      ? data.job.project_authorization_doc_hash.slice(0, 8)
                      : "No hash"}
                  </button>
                {:else if data.job.project_authorization_doc_hash}
                  <span class="font-mono text-sm opacity-70">
                    {data.job.project_authorization_doc_hash.slice(0, 8)}
                  </span>
                {/if}
                {#if data.job.project_authorization_doc_hash}
                  <button
                    type="button"
                    class="text-neutral-500 hover:text-neutral-700"
                    title="Copy full PA hash"
                    onclick={() =>
                      navigator.clipboard.writeText(data.job.project_authorization_doc_hash)}
                  >
                    <Icon icon="mdi:content-copy" width="16" />
                  </button>
                {/if}
              </div>
            {/if}
            {#if data.job.pa_reviewed && data.job.pa_reviewer?.id}
              <div>
                <span class="font-semibold">Reviewed By:</span>
                {personName(data.job.pa_reviewer)}
                <span class="text-sm text-neutral-500">
                  ({shortDate(data.job.pa_reviewed, true)})
                </span>
              </div>
            {/if}
            {#if canUploadProjectAuthorization && !projectAuthorizationApproved}
              <label class="flex max-w-sm flex-col gap-1 text-sm">
                <span class="font-semibold">Upload Signed PA PDF</span>
                <input
                  type="file"
                  accept="application/pdf"
                  disabled={paUploading}
                  onchange={uploadProjectAuthorizationDoc}
                  class="rounded-sm border border-neutral-300 p-2"
                />
              </label>
            {/if}
            {#if paUploadError}
              <div class="text-sm text-red-600">{paUploadError}</div>
            {/if}
            {#if canDeleteProjectAuthorization}
              <div>
                <DsActionButton action={() => (showPADeleteConfirm = true)} color="yellow">
                  Remove PA PDF
                </DsActionButton>
              </div>
            {/if}
            {#if canRevokeProjectAuthorization}
              <div>
                <DsActionButton action={() => (showPARevokeConfirm = true)} color="red">
                  Revoke PA Approval
                </DsActionButton>
              </div>
            {/if}
          </div>

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
        {:else}
          <div>
            <span class="font-semibold">Project Value:</span>
            {formatCurrency(data.job.project_value ?? 0)}
          </div>
        {/if}

        <div>
          <span class="font-semibold">Time & Materials:</span>
          {data.job.time_and_materials ? "Yes" : "No"}
        </div>

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
      <StaffSummaryContent jobId={data.job.id} startDate={timeRangeStart} endDate={timeRangeEnd} />
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
