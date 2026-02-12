<script lang="ts">
  import { flatpickrAction, fetchCategories, applyDefaultDivisionOnce } from "$lib/utilities";
  import { jobs } from "$lib/stores/jobs";
  import { vendors } from "$lib/stores/vendors";
  import { divisions } from "$lib/stores/divisions";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import DsFileSelect from "$lib/components/DsFileSelect.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { authStore } from "$lib/stores/auth";
  import { goto } from "$app/navigation";
  import type { PurchaseOrdersPageData } from "$lib/svelte-types";
  import type {
    CategoriesResponse,
    ExpenditureKindsResponse,
    PoApproversResponse,
  } from "$lib/pocketbase-types";
  import DsActionButton from "./DSActionButton.svelte";
  import DsLabel from "./DsLabel.svelte";
  import { untrack } from "svelte";
  import { expensesEditingEnabled } from "$lib/stores/appConfig";
  import DsEditingDisabledBanner from "./DsEditingDisabledBanner.svelte";

  // initialize the stores, noop if already initialized
  jobs.init();
  vendors.init();
  divisions.init();

  let { data }: { data: PurchaseOrdersPageData } = $props();

  let errors = $state({} as any);
  let item = $state(untrack(() => data.item));

  const isRecurring = $derived(item.type === "Recurring");
  const isChildPO = $derived(item.parent_po !== "" && item.parent_po !== undefined);

  let categories = $state([] as CategoriesResponse[]);
  let expenditureKinds = $state([] as ExpenditureKindsResponse[]);
  let approvers = $state([] as PoApproversResponse[]);
  let secondApprovers = $state([] as PoApproversResponse[]);
  let showApproverField = $state(false);
  let showSecondApproverField = $state(false);
  let loadedKinds = $state(false);

  const kindOptions = $derived.by(() =>
    expenditureKinds.map((kind) => ({
      id: kind.id,
      label: kind.en_ui_label,
    })),
  );
  const selectedKind = $derived.by(() => expenditureKinds.find((kind) => kind.id === item.kind));
  const standardKindId = $derived.by(
    () => expenditureKinds.find((kind) => kind.name === "standard")?.id ?? "",
  );
  const typeOptions = [
    { id: "One-Time", label: "One-Time" },
    { id: "Cumulative", label: "Cumulative" },
    { id: "Recurring", label: "Recurring" },
  ];
  const selectedType = $derived.by(() => typeOptions.find((type) => type.id === item.type));
  const selectedTypeDescription = $derived.by(() => {
    if (item.type === "Recurring") {
      return "for recurring expenses, requires an end date and frequency";
    }
    if (item.type === "Cumulative") {
      return "allows multiple expenses until the PO total is used";
    }
    return "for a single expense, closes after use";
  });
  const paymentTypeOptions = [
    { id: "OnAccount", label: "On Account" },
    { id: "Expense", label: "Expense" },
    { id: "CorporateCreditCard", label: "Corporate Credit Card" },
  ];
  const selectedPaymentType = $derived.by(() =>
    paymentTypeOptions.find((paymentType) => paymentType.id === item.payment_type),
  );
  const selectedPaymentDescription = $derived.by(() => {
    if (item.payment_type === "CorporateCreditCard") {
      return "use for purchases paid directly with a corporate card";
    }
    if (item.payment_type === "Expense") {
      return "use when purchases are paid out-of-pocket and reimbursed";
    }
    return "use for purchases charged directly to a vendor account";
  });

  // Watch for changes to the job and fetch categories accordingly
  $effect(() => {
    if (item.job) {
      fetchCategories(item.job).then((c) => (categories = c));
    }
  });

  // Default division from caller's profile if creating and empty
  $effect(() => applyDefaultDivisionOnce(item, data.editing));

  // Load expenditure kinds once for the kind toggle.
  $effect(() => {
    if (loadedKinds) return;
    loadedKinds = true;
    pb.collection("expenditure_kinds")
      .getFullList<ExpenditureKindsResponse>({ sort: "ui_order" })
      .then((rows) => {
        expenditureKinds = rows;
        if (!item.kind || item.kind === "") {
          const standard = rows.find((r) => r.name === "standard");
          item.kind = standard?.id ?? rows[0]?.id ?? "";
        }
      })
      .catch((error) => {
        console.error("Error loading expenditure kinds:", error);
      });
  });

  // Watch for changes to division, amount, or type to fetch approvers
  $effect(() => {
    if (item.division && item.total && item.kind) {
      fetchApprovers();
    }
  });

  // A PO with a job is always treated as project expense (standard kind + has_job=true).
  $effect(() => {
    if (item.job !== "" && standardKindId && item.kind !== standardKindId) {
      item.kind = standardKindId;
    }
  });

  async function fetchApprovers() {
    try {
      const params = new URLSearchParams({
        division: item.division,
        amount: String(Number(item.total)),
        kind: item.kind,
        has_job: String(item.job !== ""),
        type: isRecurring ? "Recurring" : item.type,
        start_date: item.date || "",
        end_date: item.end_date || "",
        frequency: item.frequency || "",
      });

      // Fetch first approvers
      approvers = await pb.send(`/api/purchase_orders/approvers?${params.toString()}`, {
        method: "GET",
      });

      // Fetch second approvers
      secondApprovers = await pb.send(
        `/api/purchase_orders/second_approvers?${params.toString()}`,
        {
          method: "GET",
        },
      );

      // Show approvers field if there are approvers available
      showApproverField = approvers.length > 0;

      // Show second approver field if there are second approvers available
      showSecondApproverField = secondApprovers.length > 0;
    } catch (error) {
      console.error("Error fetching approvers:", error);
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
      const standard = expenditureKinds.find((k) => k.name === "standard");
      item.kind = standard?.id ?? expenditureKinds[0]?.id ?? "";
    }

    // set approver to self if the approver field is hidden
    if (!showApproverField) {
      item.approver = item.uid;
    }

    // set priority_second_approver to self if the second approver field is
    // hidden. This will be cleared in the backend (cleanPurchaseOrder) if the
    // total is less than the lowest threshold.
    if (!showSecondApproverField) {
      item.priority_second_approver = item.uid;
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
  class="flex w-full flex-col items-center gap-2 p-2"
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

  <div
    class="grid w-full grid-cols-[auto_1fr] items-center gap-x-3 gap-y-1 {errors.type !== undefined
      ? 'bg-red-200'
      : ''}"
  >
    <div>
      <label for="po-type">Type</label>
    </div>
    <div>
      {#if isChildPO}
        <DsLabel color="cyan">{selectedType?.label ?? "One-Time"}</DsLabel>
      {:else}
        <DSToggle bind:value={item.type} options={typeOptions} />
      {/if}
    </div>
    <span class="col-start-2 text-sm text-neutral-600">{selectedTypeDescription}</span>
    {#if errors.type !== undefined}
      <span class="col-start-2 text-red-600">{errors.type.message}</span>
    {/if}
  </div>

  <div
    class="grid w-full grid-cols-[auto_1fr] items-center gap-x-3 gap-y-1 {errors.kind !== undefined
      ? 'bg-red-200'
      : ''}"
  >
    <div>
      <label for="po-kind">Kind</label>
    </div>
    <div>
      {#if item.job !== ""}
        <DsLabel color="cyan">{selectedKind?.en_ui_label ?? "Standard"}</DsLabel>
      {:else}
        <DSToggle bind:value={item.kind} options={kindOptions} />
      {/if}
    </div>
    {#if selectedKind && selectedKind.name !== "standard" && selectedKind.description}
      <span class="col-start-2 text-sm text-neutral-600">{selectedKind.description}</span>
    {/if}
    {#if errors.kind !== undefined}
      <span class="col-start-2 text-red-600">{errors.kind.message}</span>
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

  <div
    class="grid w-full grid-cols-[auto_1fr] items-center gap-x-3 gap-y-1 {errors.payment_type !==
    undefined
      ? 'bg-red-200'
      : ''}"
  >
    <div>
      <label for="po-payment-type">Payment</label>
    </div>
    <div>
      {#if isChildPO}
        <DsLabel color="cyan">{selectedPaymentType?.label ?? "On Account"}</DsLabel>
      {:else}
        <DSToggle bind:value={item.payment_type} options={paymentTypeOptions} />
      {/if}
    </div>
    <span class="col-start-2 text-sm text-neutral-600">{selectedPaymentDescription}</span>
    {#if errors.payment_type !== undefined}
      <span class="col-start-2 text-red-600">{errors.payment_type.message}</span>
    {/if}
  </div>

  {#if $jobs.index !== null}
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

  {#if $vendors.index !== null}
    <DsAutoComplete
      bind:value={item.vendor as string}
      index={$vendors.index}
      {errors}
      fieldName="vendor"
      uiName="Vendor"
      disabled={isChildPO}
    >
      {#snippet resultTemplate(item)}{item.name} ({item.alias}){/snippet}
    </DsAutoComplete>
  {/if}

  <!-- File upload for attachment -->
  <DsFileSelect bind:record={item} {errors} fieldName="attachment" uiName="Attachment" />

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
