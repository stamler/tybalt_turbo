<script lang="ts">
  import {
    DATE_INPUT_MIN,
    applyDefaultDivisionOnce,
    createJobCategoriesSync,
    dateInputMaxMonthsAhead,
  } from "$lib/utilities";
  import { expenditureKinds as expenditureKindsStore } from "$lib/stores/expenditureKinds";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsDateInput from "$lib/components/DSDateInput.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsFileSelect from "$lib/components/DsFileSelect.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { authStore } from "$lib/stores/auth";
  import { divisions } from "$lib/stores/divisions";
  import { goto } from "$app/navigation";
  import type { ExpensesPageData } from "$lib/svelte-types";
  import type { CategoriesResponse, ExpensesAllowanceTypesOptions } from "$lib/pocketbase-types";
  import { isExpensesResponse } from "$lib/pocketbase-types";
  import DsActionButton from "./DSActionButton.svelte";
  import CumulativePOOverflowModal from "./CumulativePOOverflowModal.svelte";
  import VendorSelector from "./VendorSelector.svelte";
  import { jobs } from "$lib/stores/jobs";
  import { untrack } from "svelte";
  import { expensesEditingEnabled } from "$lib/stores/appConfig";
  import DsEditingDisabledBanner from "./DsEditingDisabledBanner.svelte";

  // initialize the stores, noop if already initialized
  jobs.init();
  divisions.init();
  expenditureKindsStore.init();
  let { data }: { data: ExpensesPageData } = $props();

  let errors = $state({} as any);
  let item = $state(untrack(() => data.item));
  const dateInputMax = dateInputMaxMonthsAhead(15);
  let overflowModal: CumulativePOOverflowModal;

  let categories = $state([] as CategoriesResponse[]);
  const syncCategoriesForJob = createJobCategoriesSync((rows) => {
    categories = rows;
  });

  const selectedKindLabel = $derived.by(() => {
    const match = $expenditureKindsStore.items.find((k) => k.id === item.kind);
    if (match) return match.en_ui_label;
    // Fallback for blank kind (shouldn't happen after migration): derive from job presence.
    if (!item.purchase_order) {
      const hasJob = item.job && item.job !== "";
      const fallbackName = hasJob ? "project" : "capital";
      return (
        $expenditureKindsStore.items.find((k) => k.name === fallbackName)?.en_ui_label ?? "Unknown"
      );
    }
    return "Unknown";
  });
  const linkedPurchaseOrderNumber = $derived.by(() => {
    if (data.linked_purchase_order?.po_number) {
      return data.linked_purchase_order.po_number;
    }
    if (isExpensesResponse(item) && item.purchase_order !== "") {
      return item.expand.purchase_order.po_number;
    }
    return "";
  });
  const linkedPurchaseOrderType = $derived.by(() => {
    if (data.linked_purchase_order?.type) {
      return data.linked_purchase_order.type;
    }
    if (isExpensesResponse(item) && item.purchase_order !== "") {
      return item.expand.purchase_order.type;
    }
    return "";
  });
  const linkedRecurringRemainingOccurrences = $derived.by(
    () => data.linked_purchase_order?.recurring_remaining_occurrences ?? null,
  );
  const linkedCumulativeRemainingBalance = $derived.by(
    () => data.linked_purchase_order?.cumulative_remaining_balance ?? null,
  );
  const authUserID = $derived.by(() => $authStore?.model?.id ?? "");
  const nonOwnerEditMessage = "You can view this expense, but only its creator can edit it.";
  const isEditingAnotherUsersExpense = $derived.by(
    () =>
      data.editing &&
      authUserID !== "" &&
      item.uid !== "" &&
      item.uid !== authUserID,
  );

  // create a local state object to hold the allowance types
  const allowanceTypes = $state({
    Breakfast: false,
    Lunch: false,
    Dinner: false,
    Lodging: false,
  });

  // Watch for changes to the job and fetch categories accordingly
  $effect(() => {
    syncCategoriesForJob(item.job);
  });

  // Default division from caller's profile if creating and empty
  $effect(() => applyDefaultDivisionOnce(item, data.editing));

  async function save(event: Event) {
    event.preventDefault();
    if (isEditingAnotherUsersExpense) {
      errors = {
        ...errors,
        global: {
          code: "not_owner",
          message: nonOwnerEditMessage,
        },
      };
      return;
    }
    item.uid = $authStore?.model?.id ?? "";

    // set a dummy value for week_ending to satisfy the schema non-empty
    // requirement. This will be changed in the backend to the correct
    // value every time a record is saved
    item.pay_period_ending = "2006-01-02";

    // if the job is empty, set the category to empty
    if (item.job === "") {
      item.category = "";
    }
    // if the payment_type is not CorporateCreditCard, then the cc_last_4_digits
    // should be empty
    if (item.payment_type !== "CorporateCreditCard") {
      item.cc_last_4_digits = "";
    }

    try {
      if (data.editing && data.id !== null) {
        await pb.collection("expenses").update(data.id, item);
      } else {
        await pb.collection("expenses").create(item);
      }

      errors = {};
      goto("/expenses/list");
    } catch (error: any) {
      // Check if this is a cumulative PO overflow error
      if (error.data?.data?.total?.code === "cumulative_po_overflow") {
        // Show the child PO creation modal populated with relevant data
        const errorData = error.data.data.total.data;
        overflowModal?.openModal({
          parent_po: errorData.purchase_order,
          parent_po_number: errorData.po_number,
          po_total: errorData.po_total,
          overflow_amount: parseFloat(errorData.overflow_amount),
        });
      } else {
        errors = error.data.data;
      }
    }
  }
</script>

<CumulativePOOverflowModal bind:this={overflowModal} />

{#if !$expensesEditingEnabled}
  <DsEditingDisabledBanner
    message="Expense editing is currently disabled during a system transition."
  />
{/if}

<form
  class="flex w-full flex-col items-center gap-2 p-2 max-lg:[&_button]:text-base max-lg:[&_input]:text-base max-lg:[&_label]:text-base max-lg:[&_select]:text-base max-lg:[&_textarea]:text-base"
  enctype="multipart/form-data"
  onsubmit={save}
>
  <h1 class="w-full text-xl font-bold text-neutral-800">
    {#if data.editing}
      Editing Expense
    {:else}
      Create Expense
    {/if}
  </h1>

  {#if isEditingAnotherUsersExpense}
    <div class="w-full rounded-sm border border-amber-300 bg-amber-100 p-2 text-sm text-amber-900">
      {nonOwnerEditMessage}
    </div>
  {/if}

  <span class="flex w-full gap-2 {errors.date !== undefined ? 'bg-red-200' : ''}">
    <label for="date">Date</label>
    <DsDateInput
      class="flex-1"
      name="date"
      min={DATE_INPUT_MIN}
      max={dateInputMax}
      bind:value={item.date}
    />
    {#if errors.date !== undefined}
      <span class="text-red-600">{errors.date.message}</span>
    {/if}
  </span>

  {#if item.purchase_order !== "" && linkedPurchaseOrderNumber}
    <span class="flex w-full gap-2">
      <DsLabel color="cyan">PO {linkedPurchaseOrderNumber}</DsLabel>
    </span>
  {/if}
  {#if item.purchase_order !== "" && linkedPurchaseOrderType === "One-Time"}
    <span
      class="w-full rounded-sm border border-amber-300 bg-amber-50 px-2 py-1 text-sm text-amber-900"
    >
      This PO will be closed after this expense is committed.
    </span>
  {/if}
  {#if item.purchase_order !== "" && linkedPurchaseOrderType === "Recurring" && linkedRecurringRemainingOccurrences !== null}
    <span
      class="w-full rounded-sm border border-cyan-300 bg-cyan-50 px-2 py-1 text-sm text-cyan-900"
    >
      {linkedRecurringRemainingOccurrences} recurrence{linkedRecurringRemainingOccurrences === 1
        ? ""
        : "s"} remaining before this PO closes.
    </span>
  {/if}
  {#if item.purchase_order !== "" && linkedPurchaseOrderType === "Cumulative" && linkedCumulativeRemainingBalance !== null}
    <span
      class="w-full rounded-sm border border-cyan-300 bg-cyan-50 px-2 py-1 text-sm text-cyan-900"
    >
      Remaining cumulative balance: ${linkedCumulativeRemainingBalance.toFixed(2)}
    </span>
  {/if}

  <div class="flex w-full flex-col gap-1 {errors.kind !== undefined ? 'bg-red-200' : ''}">
    <div class="flex items-center gap-3">
      <label for="expense-kind">Kind</label>
      <DsLabel color="cyan">{selectedKindLabel}</DsLabel>
    </div>
    {#if errors.kind !== undefined}
      <span class="text-red-600">{errors.kind.message}</span>
    {/if}
  </div>

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

  <DsSelector
    bind:value={item.payment_type as string}
    items={[
      { id: "OnAccount", name: "On Account" },
      { id: "Expense", name: "Expense" },
      { id: "CorporateCreditCard", name: "Corporate Credit Card" },
      { id: "Allowance", name: "Allowance" },
      { id: "FuelCard", name: "Fuel Card" },
      { id: "Mileage", name: "Mileage" },
      { id: "PersonalReimbursement", name: "Personal Reimbursement" },
    ]}
    {errors}
    fieldName="payment_type"
    uiName="Payment Type"
  >
    {#snippet optionTemplate(item)}
      {item.name}
    {/snippet}
  </DsSelector>

  <!-- Job + category visibility rules:
   if a purchase order is present, keep Job locked to the PO but allow Category edits.
   Without a purchase order, show/edit these fields only for Allowance, Mileage,
   FuelCard, or PersonalReimbursement.
   -->
  {#if $jobs.index !== null}
    <DsAutoComplete
      bind:value={item.job as string}
      index={$jobs.index}
      {errors}
      fieldName="job"
      uiName="Job"
      disabled={item.purchase_order !== "" ||
        !(
          item.payment_type === "Allowance" ||
          item.payment_type === "Mileage" ||
          item.payment_type === "FuelCard" ||
          item.payment_type === "PersonalReimbursement"
        )}
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
    >
      {#snippet optionTemplate(item: CategoriesResponse)}
        {item.name}
      {/snippet}
    </DsSelector>
  {/if}

  {#if item.payment_type === "Allowance"}
    <div
      class="flex w-full flex-col gap-2 {errors['allowance_types'] !== undefined
        ? 'bg-red-200'
        : ''}"
    >
      <span class="flex w-full gap-4">
        <label for="allowanceTypes">Type</label>
        {#each Object.keys(allowanceTypes) as type}
          <span class="flex items-center gap-1">
            <input
              type="checkbox"
              checked={item.allowance_types.includes(type as ExpensesAllowanceTypesOptions)}
              onchange={(e) => {
                if ((e.target as HTMLInputElement).checked) {
                  item.allowance_types = [
                    ...item.allowance_types,
                    type as ExpensesAllowanceTypesOptions,
                  ];
                } else {
                  item.allowance_types = item.allowance_types.filter((t) => t !== type);
                }
              }}
            />
            {type}
          </span>
        {/each}
      </span>
      {#if errors["allowance_types"] !== undefined}
        <span class="text-red-600">{errors["allowance_types"].message}</span>
      {/if}
    </div>
  {:else}
    {#if item.payment_type === "CorporateCreditCard"}
      <DsTextInput
        bind:value={item.cc_last_4_digits as string}
        {errors}
        fieldName="cc_last_4_digits"
        uiName="Credit Card"
        placeholder="Last 4 Digits of Corporate Credit Card"
      />
    {/if}

    <DsTextInput
      bind:value={item.description as string}
      {errors}
      fieldName="description"
      uiName="Description"
    />

    {#if item.payment_type !== "Mileage"}
      <DsTextInput
        bind:value={item.total as number}
        {errors}
        fieldName="total"
        uiName="Total"
        type="number"
        step={0.01}
        min={0}
      />
      <VendorSelector bind:value={item.vendor as string} {errors} />
    {:else}
      <DsTextInput
        bind:value={item.distance as number}
        {errors}
        fieldName="distance"
        uiName="Distance"
        placeholder="in kilometers"
        type="number"
        step={1}
        min={1}
      />
    {/if}

    <!-- File upload for attachment -->
    <DsFileSelect bind:record={item} {errors} fieldName="attachment" uiName="Attachment" />
  {/if}

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      {#if !isEditingAnotherUsersExpense}
        <DsActionButton type="submit">Save</DsActionButton>
      {/if}
      <DsActionButton action="/expenses/list"
        >{isEditingAnotherUsersExpense ? "Back" : "Cancel"}</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
