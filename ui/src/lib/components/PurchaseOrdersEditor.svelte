<script lang="ts">
  import {
    flatpickrAction,
    applyDefaultDivisionOnce,
    createJobCategoriesSync,
  } from "$lib/utilities";
  import { jobs } from "$lib/stores/jobs";
  import { divisions } from "$lib/stores/divisions";
  import { branches as branchesStore } from "$lib/stores/branches";
  import { expenditureKinds as expenditureKindsStore } from "$lib/stores/expenditureKinds";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import DsFileSelect from "$lib/components/DsFileSelect.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { authStore } from "$lib/stores/auth";
  import { goto } from "$app/navigation";
  import type {
    PurchaseOrdersPageData,
    SecondApproversResponse,
    SecondApproverStatus,
  } from "$lib/svelte-types";
  import type {
    BranchesResponse,
    CategoriesResponse,
    PoApproversResponse,
  } from "$lib/pocketbase-types";
  import {
    buildPoApproverRequest,
    fetchPoApproversBundle,
    type PoApproverRequest,
  } from "$lib/poApprovers";
  import DsActionButton from "./DSActionButton.svelte";
  import DsLabel from "./DsLabel.svelte";
  import PoSecondApproverStatus from "./PoSecondApproverStatus.svelte";
  import VendorSelector from "./VendorSelector.svelte";
  import { untrack } from "svelte";
  import { expensesEditingEnabled } from "$lib/stores/appConfig";
  import { globalStore } from "$lib/stores/global";
  import DsEditingDisabledBanner from "./DsEditingDisabledBanner.svelte";

  // initialize the stores, noop if already initialized
  jobs.init();
  divisions.init();
  branchesStore.init();
  expenditureKindsStore.init();

  let { data }: { data: PurchaseOrdersPageData } = $props();

  let errors = $state({} as any);
  let item = $state(untrack(() => data.item));

  const isRecurring = $derived(item.type === "Recurring");
  const isChildPO = $derived(item.parent_po !== "" && item.parent_po !== undefined);

  let categories = $state([] as CategoriesResponse[]);
  const syncCategoriesForJob = createJobCategoriesSync((rows) => {
    categories = rows;
  });
  let approvers = $state([] as PoApproversResponse[]);
  let secondApprovers = $state([] as PoApproversResponse[]);
  let showApproverField = $state(false);
  let showSecondApproverField = $state(false);
  let secondApproverStatus = $state("" as SecondApproverStatus | "");
  let secondApproverReasonMessage = $state("");
  let secondApproverMeta = $state<SecondApproversResponse["meta"] | null>(null);
  let approversLoaded = $state(false);
  let approversFetchError = $state(false);
  let approverFetchRequestId = 0;
  let stashedJobWhenKindDisallows = $state("");
  let branchPinnedInSession = $state(false);
  let lastAutoBranch = $state("");
  let branchChangeWatchInitialized = $state(false);
  let lastObservedJobForAuto = $state<string | null>(null);
  let branchLookupRequestId = 0;
  type MaybeAbortError = {
    isAbort?: boolean;
    originalError?: { name?: string };
    message?: string;
    status?: number;
  };
  const kindOptions = $derived.by(() =>
    $expenditureKindsStore.items.map((kind) => ({
      id: kind.id,
      label: kind.en_ui_label,
      description: kind.description ?? "",
    })),
  );
  const selectedKind = $derived.by(() =>
    $expenditureKindsStore.items.find((kind) => kind.id === item.kind),
  );
  const kindAllowsJob = $derived.by(() => selectedKind?.allow_job ?? true);
  const typeOptions = [
    {
      id: "One-Time",
      label: "One-Time",
      description: "for a single expense, closes after use",
    },
    {
      id: "Cumulative",
      label: "Cumulative",
      description: "allows multiple expenses until the PO total is used",
    },
    {
      id: "Recurring",
      label: "Recurring",
      description: "for recurring expenses, requires an end date and frequency",
    },
  ];
  const selectedType = $derived.by(() => typeOptions.find((type) => type.id === item.type));
  const paymentTypeOptions = [
    {
      id: "OnAccount",
      label: "On Account",
      description: "use for purchases charged directly to a vendor account",
    },
    {
      id: "Expense",
      label: "Expense",
      description: "use when purchases are paid out-of-pocket and reimbursed",
    },
    {
      id: "CorporateCreditCard",
      label: "Corporate Credit Card",
      description: "use for purchases paid directly with a corporate card",
    },
  ];
  const selectedPaymentType = $derived.by(() =>
    paymentTypeOptions.find((paymentType) => paymentType.id === item.payment_type),
  );
  const creatorDefaultBranch = $derived.by(() => $globalStore.profile.default_branch ?? "");
  const approverRequest = $derived.by(() =>
    buildPoApproverRequest({
      division: item.division,
      total: item.total,
      kind: item.kind,
      job: item.job,
      type: isRecurring ? "Recurring" : item.type,
      date: item.date,
      end_date: item.end_date,
      frequency: item.frequency,
    }),
  );
  const canFetchApprovers = $derived.by(() => Boolean(item.division && item.total && item.kind));
  const showApproverFetchError = $derived.by(() => canFetchApprovers && approversFetchError);
  const showSecondApproverStatusHint = $derived.by(
    () => approversLoaded && canFetchApprovers && !approversFetchError && !showSecondApproverField,
  );
  const isAbortError = (error: unknown): boolean => {
    const e = error as MaybeAbortError;
    if (e?.isAbort) return true;
    if (e?.originalError?.name === "AbortError") return true;
    return e?.status === 0 && (e?.message ?? "").toLowerCase().includes("aborted");
  };

  async function resolveDerivedBranch(
    jobId: string,
    fallbackDefaultBranch: string,
  ): Promise<string> {
    if (jobId !== "") {
      try {
        const job = await pb.collection("jobs").getOne(jobId, { requestKey: null });
        return job.branch ?? "";
      } catch (error) {
        console.error("Error loading job branch:", error);
      }
    }
    return fallbackDefaultBranch;
  }

  $effect(() => {
    const branch = item.branch ?? "";
    if ((item.job ?? "") !== "") {
      return;
    }
    if (!branchChangeWatchInitialized) {
      branchChangeWatchInitialized = true;
      return;
    }
    if (branch === "" || branch === lastAutoBranch) {
      return;
    }
    branchPinnedInSession = true;
  });

  $effect(() => {
    const jobId = item.job ?? "";
    const fallbackDefaultBranch = creatorDefaultBranch;
    const branch = item.branch ?? "";
    const pinned = branchPinnedInSession;

    if (jobId !== "") {
      branchPinnedInSession = false;
      lastObservedJobForAuto = jobId;
      const requestId = ++branchLookupRequestId;
      resolveDerivedBranch(jobId, fallbackDefaultBranch).then((derivedBranch) => {
        if (requestId !== branchLookupRequestId || derivedBranch === "") {
          return;
        }
        if (item.branch === derivedBranch) {
          return;
        }
        lastAutoBranch = derivedBranch;
        item.branch = derivedBranch;
      });
      return;
    }

    if (pinned) {
      lastObservedJobForAuto = jobId;
      return;
    }

    const firstObservation = lastObservedJobForAuto === null;
    const jobChanged = !firstObservation && lastObservedJobForAuto !== jobId;
    lastObservedJobForAuto = jobId;

    // Auto-populate when branch starts empty, and auto-switch on job changes
    // while the branch remains unpinned in this editor session.
    if (!jobChanged && branch !== "") {
      return;
    }

    const requestId = ++branchLookupRequestId;
    resolveDerivedBranch(jobId, fallbackDefaultBranch).then((derivedBranch) => {
      if (requestId !== branchLookupRequestId || derivedBranch === "") {
        return;
      }
      if (item.branch === derivedBranch) {
        return;
      }
      lastAutoBranch = derivedBranch;
      item.branch = derivedBranch;
    });
  });

  // Watch for changes to the job and fetch categories accordingly
  $effect(() => {
    syncCategoriesForJob(item.job);
  });

  // Default division from caller's profile if creating and empty
  $effect(() => applyDefaultDivisionOnce(item, data.editing));

  // Default kind once kinds are loaded.
  $effect(() => {
    const kinds = $expenditureKindsStore.items;
    if (kinds.length === 0) return;
    if (!item.kind || item.kind === "") {
      const standard = kinds.find((r) => r.name === "standard");
      item.kind = standard?.id ?? kinds[0]?.id ?? "";
    }
  });

  // Keep approver options in sync with all request parameters used by
  // /api/purchase_orders/approvers and /second_approvers.
  $effect(() => {
    const request = approverRequest;
    if (canFetchApprovers) {
      fetchApprovers(request);
    } else {
      approverFetchRequestId += 1;
      approvers = [];
      secondApprovers = [];
      showApproverField = false;
      showSecondApproverField = false;
      secondApproverStatus = "";
      secondApproverReasonMessage = "";
      secondApproverMeta = null;
      approversLoaded = false;
      approversFetchError = false;
    }
  });

  // Kind wins over job. If selected kind disallows jobs, stash+clear job and
  // restore it when switching back to a kind that allows jobs.
  $effect(() => {
    if (kindAllowsJob) {
      if (item.job === "" && stashedJobWhenKindDisallows !== "") {
        item.job = stashedJobWhenKindDisallows;
        stashedJobWhenKindDisallows = "";
      }
      return;
    }
    if (item.job !== "") {
      stashedJobWhenKindDisallows = item.job;
      item.job = "";
    }
  });

  async function fetchApprovers(args: PoApproverRequest) {
    const requestId = ++approverFetchRequestId;
    approversFetchError = false;
    try {
      const { approvers: nextApprovers, secondApproversResponse } = await fetchPoApproversBundle(
        args,
        null, // Avoid PocketBase auto-cancel while users type quickly.
      );
      if (requestId !== approverFetchRequestId) return;
      approvers = nextApprovers;
      secondApprovers = secondApproversResponse.approvers;
      secondApproverStatus = secondApproversResponse.meta.status;
      secondApproverReasonMessage = secondApproversResponse.meta.reason_message ?? "";
      secondApproverMeta = secondApproversResponse.meta;

      // Show approvers field if there are approvers available
      showApproverField = approvers.length > 0;

      // Show second approver selector only when candidates are available.
      showSecondApproverField =
        secondApproverStatus === "candidates_available" && secondApprovers.length > 0;
      approversLoaded = true;
      approversFetchError = false;
    } catch (error) {
      if (requestId !== approverFetchRequestId) return;
      if (isAbortError(error)) return;
      console.error("Error fetching approvers:", error);
      approvers = [];
      secondApprovers = [];
      showApproverField = false;
      showSecondApproverField = false;
      secondApproverStatus = "";
      secondApproverReasonMessage = "";
      secondApproverMeta = null;
      approversLoaded = false;
      approversFetchError = true;
    }
  }

  async function save(event: Event) {
    event.preventDefault();
    item.uid = $authStore?.model?.id ?? "";

    // if the job is empty, set the category to empty
    if (item.job === "") {
      item.category = "";
    }
    if (!item.kind || item.kind === "") {
      const standard = $expenditureKindsStore.items.find((k) => k.name === "standard");
      item.kind = standard?.id ?? $expenditureKindsStore.items[0]?.id ?? "";
    }
    if (!kindAllowsJob) {
      item.job = "";
      item.category = "";
    }

    // set approver to self if the approver field is hidden
    if (!showApproverField) {
      item.approver = item.uid;
    }

    // Set priority_second_approver based on second-approval status when the
    // selector is hidden.
    if (!showSecondApproverField) {
      if (
        secondApproverStatus === "not_required" ||
        secondApproverStatus === "requester_qualifies"
      ) {
        item.priority_second_approver = item.uid;
      } else if (secondApproverStatus === "required_no_candidates") {
        item.priority_second_approver = "";
      }
    }

    try {
      if (data.editing && data.id !== null) {
        await pb.collection("purchase_orders").update(data.id, item);
      } else {
        await pb.collection("purchase_orders").create(item);
      }

      errors = {};
      goto("/pos/list");
    } catch (error: any) {
      errors = error.data.data;
    }
  }
</script>

<svelte:head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css" />
</svelte:head>

{#if !$expensesEditingEnabled}
  <DsEditingDisabledBanner
    message="Purchase order editing is currently disabled during a system transition."
  />
{/if}

<form
  class="flex w-full flex-col items-center gap-2 p-2 max-lg:[&_button]:text-base max-lg:[&_input]:text-base max-lg:[&_label]:text-base max-lg:[&_select]:text-base max-lg:[&_textarea]:text-base"
  enctype="multipart/form-data"
  onsubmit={save}
>
  <h1 class="w-full text-xl font-bold text-neutral-800">
    {#if data.editing}
      {#if item.po_number}
        Editing {item.po_number}
      {:else}
        Editing Purchase Order
      {/if}
    {:else}
      Create Purchase Order
    {/if}
  </h1>

  {#if isChildPO && data.parent_po_number}
    <span class="flex w-full gap-2 {errors.parent_po !== undefined ? 'bg-red-200' : ''}">
      <DsLabel color="cyan">Child PO of {data.parent_po_number}</DsLabel>
      {#if errors.parent_po !== undefined}
        <span class="text-red-600">{errors.parent_po.message}</span>
      {/if}
    </span>
  {/if}

  <div class="flex w-full flex-col gap-1 {errors.type !== undefined ? 'bg-red-200' : ''}">
    {#if isChildPO}
      <span>Type</span>
      <DsLabel color="cyan">{selectedType?.label ?? "One-Time"}</DsLabel>
    {:else}
      <DSToggle
        bind:value={item.type}
        label="Type"
        options={typeOptions}
        showOptionDescriptions={true}
        fullWidth={true}
      />
    {/if}
    {#if errors.type !== undefined}
      <span class="text-red-600">{errors.type.message}</span>
    {/if}
  </div>

  <div class="flex w-full flex-col gap-1 {errors.kind !== undefined ? 'bg-red-200' : ''}">
    <DSToggle
      bind:value={item.kind}
      label="Kind"
      options={kindOptions}
      showOptionDescriptions={true}
      fullWidth={true}
    />
    {#if errors.kind !== undefined}
      <span class="text-red-600">{errors.kind.message}</span>
    {/if}
  </div>

  {#if showApproverField}
    <DsSelector
      bind:value={item.approver as string}
      items={approvers}
      {errors}
      fieldName="approver"
      uiName="Approver"
    >
      {#snippet optionTemplate(item)}
        {item.given_name} {item.surname}
      {/snippet}
    </DsSelector>
  {/if}

  {#if showSecondApproverField}
    <DsSelector
      bind:value={item.priority_second_approver as string}
      items={secondApprovers}
      {errors}
      fieldName="priority_second_approver"
      uiName="Priority Second Approver"
      clear={true}
    >
      {#snippet optionTemplate(item)}
        {item.given_name} {item.surname}
      {/snippet}
    </DsSelector>
  {:else}
    <PoSecondApproverStatus
      showFetchError={showApproverFetchError}
      showStatusHint={showSecondApproverStatusHint}
      status={secondApproverStatus}
      reasonMessage={secondApproverReasonMessage}
      meta={secondApproverMeta}
      division={item.division ?? ""}
      kindLabel={selectedKind?.en_ui_label ?? item.kind ?? "n/a"}
      hasJob={item.job !== ""}
    />
  {/if}

  {#if isRecurring}
    <span class="flex w-full gap-2 {errors.end_date !== undefined ? 'bg-red-200' : ''}">
      <label for="end_date">End Date</label>
      <input
        class="flex-1"
        type="text"
        name="end_date"
        placeholder="End Date"
        use:flatpickrAction
        bind:value={item.end_date}
      />
      {#if errors.end_date !== undefined}
        <span class="text-red-600">{errors.end_date.message}</span>
      {/if}
    </span>

    <DsSelector
      bind:value={item.frequency as string}
      items={[
        { id: "Weekly", name: "Weekly" },
        { id: "Biweekly", name: "Biweekly" },
        { id: "Monthly", name: "Monthly" },
      ]}
      {errors}
      fieldName="frequency"
      uiName="Frequency"
    >
      {#snippet optionTemplate(item)}
        {item.name}
      {/snippet}
    </DsSelector>
  {/if}

  <span class="flex w-full gap-2 {errors.date !== undefined ? 'bg-red-200' : ''}">
    <label for="date">Date</label>
    <input
      class="flex-1"
      type="text"
      name="date"
      placeholder="Date"
      use:flatpickrAction
      bind:value={item.date}
    />
    {#if errors.date !== undefined}
      <span class="text-red-600">{errors.date.message}</span>
    {/if}
  </span>

  {#if $divisions.index !== null}
    <DsAutoComplete
      bind:value={item.division as string}
      index={$divisions.index}
      {errors}
      fieldName="division"
      uiName="Division"
    >
      {#snippet resultTemplate(item)}{item.code} - {item.name}{/snippet}
    </DsAutoComplete>
  {/if}

  {#if item.job === ""}
    <DsSelector
      bind:value={item.branch as string}
      items={$branchesStore.items}
      {errors}
      fieldName="branch"
      uiName="Branch"
    >
      {#snippet optionTemplate(item: BranchesResponse)}
        {item.name}
      {/snippet}
    </DsSelector>
  {/if}

  <div class="flex w-full flex-col gap-1 {errors.payment_type !== undefined ? 'bg-red-200' : ''}">
    {#if isChildPO}
      <span>Payment</span>
      <DsLabel color="cyan">{selectedPaymentType?.label ?? "On Account"}</DsLabel>
    {:else}
      <DSToggle
        bind:value={item.payment_type}
        label="Payment"
        options={paymentTypeOptions}
        showOptionDescriptions={true}
        fullWidth={true}
      />
    {/if}
    {#if errors.payment_type !== undefined}
      <span class="text-red-600">{errors.payment_type.message}</span>
    {/if}
  </div>

  {#if kindAllowsJob && $jobs.index !== null}
    <DsAutoComplete
      bind:value={item.job as string}
      index={$jobs.index}
      {errors}
      fieldName="job"
      uiName="Job"
      disabled={isChildPO}
    >
      {#snippet resultTemplate(item)}{item.number} - {item.description}{/snippet}
    </DsAutoComplete>
  {/if}

  {#if item.job !== "" && categories.length > 0}
    <DsSelector
      bind:value={item.category as string}
      items={categories}
      {errors}
      fieldName="category"
      uiName="Category"
      clear={true}
      disabled={isChildPO}
    >
      {#snippet optionTemplate(item: CategoriesResponse)}
        {item.name}
      {/snippet}
    </DsSelector>
  {/if}

  <DsTextInput
    bind:value={item.description as string}
    {errors}
    fieldName="description"
    uiName="Description"
    disabled={isChildPO}
  />

  <DsTextInput
    bind:value={item.total as number}
    {errors}
    fieldName="total"
    uiName="Total"
    type="number"
    step={0.01}
    min={0}
  />
  <span class="flex w-full gap-2 text-sm text-neutral-600">
    <span class="invisible">Total</span>
    <span>max in CAD including all taxes and shipping</span>
  </span>

  <VendorSelector bind:value={item.vendor as string} {errors} disabled={isChildPO} />

  <!-- File upload for attachment -->
  <DsFileSelect bind:record={item} {errors} fieldName="attachment" uiName="Attachment" />
  <span class="flex w-full gap-2 text-sm text-neutral-600">
    <span class="invisible">Attachment</span>
    <span>include quotes, agreements, or any relevant supporting documentation</span>
  </span>

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton type="submit">Save</DsActionButton>
      <DsActionButton action="/pos/list">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
