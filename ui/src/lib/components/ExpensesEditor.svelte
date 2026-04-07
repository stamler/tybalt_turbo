<script lang="ts">
  import {
    DATE_INPUT_MIN,
    applyDefaultDivisionOnce,
    createJobCategoriesSync,
    dateInputMaxMonthsAhead,
  } from "$lib/utilities";
  import { expenditureKinds as expenditureKindsStore } from "$lib/stores/expenditureKinds";
  import { currencies } from "$lib/stores/currencies";
  import { pb } from "$lib/pocketbase";
  import DSCurrencyInput from "$lib/components/DSCurrencyInput.svelte";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsDateInput from "$lib/components/DSDateInput.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsFileSelect from "$lib/components/DsFileSelect.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { authStore } from "$lib/stores/auth";
  import { divisions } from "$lib/stores/divisions";
  import { goto } from "$app/navigation";
  import { resolve } from "$app/paths";
  import type { ExpensesPageData } from "$lib/svelte-types";
  import type { CategoriesResponse, ExpensesAllowanceTypesOptions } from "$lib/pocketbase-types";
  import { isExpensesResponse } from "$lib/pocketbase-types";
  import DsActionButton from "./DSActionButton.svelte";
  import CumulativePOOverflowModal from "./CumulativePOOverflowModal.svelte";
  import VendorSelector from "./VendorSelector.svelte";
  import { jobs } from "$lib/stores/jobs";
  import { untrack } from "svelte";
  import { expensesEditingEnabled } from "$lib/stores/appConfig";
  import { globalStore } from "$lib/stores/global";
  import DsEditingDisabledBanner from "./DsEditingDisabledBanner.svelte";

  // initialize the stores, noop if already initialized
  jobs.init();
  divisions.init();
  currencies.init();
  expenditureKindsStore.init();
  let { data }: { data: ExpensesPageData } = $props();

  let errors = $state({} as Record<string, { message: string; code?: string }>);
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
  const linkedRemainingAmount = $derived.by(
    () => data.linked_purchase_order?.remaining_amount ?? null,
  );
  const authUserID = $derived.by(() => $authStore?.model?.id ?? "");
  const selectedCurrency = $derived.by(() => $currencies.items.find((row) => row.id === item.currency));
  const selectedCurrencyCode = $derived.by(() => selectedCurrency?.code ?? "CAD");
  const homeCurrency = $derived.by(() => $currencies.items.find((row) => row.code === "CAD"));
  const fixedHomePaymentType = $derived.by(
    () =>
      item.payment_type === "Allowance" ||
      item.payment_type === "FuelCard" ||
      item.payment_type === "Mileage" ||
      item.payment_type === "PersonalReimbursement",
  );
  const linkedPurchaseOrderCurrency = $derived.by(() => {
    if (data.linked_purchase_order?.currency) {
      return data.linked_purchase_order.currency;
    }
    if (isExpensesResponse(item) && item.purchase_order !== "") {
      return item.expand.purchase_order.currency ?? "";
    }
    return "";
  });
  const currencySelectionDisabled = $derived.by(
    () => item.purchase_order !== "" || fixedHomePaymentType,
  );
  const showSettledTotalInput = $derived.by(
    () => selectedCurrencyCode !== "CAD" && item.payment_type === "Expense",
  );
  const nonOwnerEditMessage = "You can view this expense, but only its creator can edit it.";
  const isEditingAnotherUsersExpense = $derived.by(
    () => data.editing && authUserID !== "" && item.uid !== "" && item.uid !== authUserID,
  );

  const allPaymentTypes = [
    { id: "OnAccount", name: "On Account" },
    { id: "Expense", name: "Expense" },
    { id: "CorporateCreditCard", name: "Corporate Credit Card" },
    { id: "Allowance", name: "Allowance" },
    { id: "FuelCard", name: "Fuel Card" },
    { id: "Mileage", name: "Mileage" },
    { id: "PersonalReimbursement", name: "Personal Reimbursement" },
  ];
  const paymentTypeOptions = $derived.by(() => {
    if (
      $globalStore.allow_personal_reimbursement.value ||
      item.payment_type === "PersonalReimbursement"
    ) {
      return allPaymentTypes;
    }
    return allPaymentTypes.filter((t) => t.id !== "PersonalReimbursement");
  });

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

  $effect(() => {
    if (item.purchase_order !== "") {
      if (linkedPurchaseOrderCurrency !== "" && item.currency !== linkedPurchaseOrderCurrency) {
        item.currency = linkedPurchaseOrderCurrency;
      }
      return;
    }

    if (item.currency !== "") {
      return;
    }

    // New records should persist an explicit CAD relation once currencies are
    // loaded, while legacy edited rows may continue to rely on blank=CAD.
    if (!data.editing && homeCurrency?.id) {
      item.currency = homeCurrency.id;
      return;
    }

    if (!currencySelectionDisabled) {
      return;
    }
    item.currency = homeCurrency?.id ?? "";
  });

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

    // pay_period_ending is server-managed and must not be sent from the editor.
    const payload: Partial<typeof item> = { ...item };
    delete payload.pay_period_ending;

    // if the job is empty, set the category to empty
    if (payload.job === "") {
      payload.category = "";
    }
    // if the payment_type is not CorporateCreditCard, then the cc_last_4_digits
    // should be empty
    if (payload.payment_type !== "CorporateCreditCard") {
      payload.cc_last_4_digits = "";
    }

    try {
      if (data.editing && data.id !== null) {
        await pb.collection("expenses").update(data.id, payload);
      } else {
        await pb.collection("expenses").create(payload);
      }

      errors = {};
      goto(resolve("/expenses/list"));
    } catch (err: unknown) {
      const error = err as {
        data?: {
          data?: Record<string, { message: string; code?: string; data?: Record<string, string> }>;
        };
      };
      // Check if this is a cumulative PO overflow error
      if (error.data?.data?.total?.code === "cumulative_po_overflow") {
        // Show the child PO creation modal populated with relevant data
        const errorData = error.data.data.total.data!;
        overflowModal?.openModal({
          parent_po: errorData.purchase_order,
          parent_po_number: errorData.po_number,
          po_total: parseFloat(errorData.po_total),
          overflow_amount: parseFloat(errorData.overflow_amount),
        });
      } else {
        errors = error.data?.data ?? {};
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
      This PO will be closed after any expense is committed.
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
  {#if item.purchase_order !== "" && linkedPurchaseOrderType === "Cumulative" && linkedRemainingAmount !== null}
    <span
      class="w-full rounded-sm border border-cyan-300 bg-cyan-50 px-2 py-1 text-sm text-cyan-900"
    >
      Provisional Remaining: ${linkedRemainingAmount.toFixed(2)} (includes uncommitted expenses)
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
    items={paymentTypeOptions}
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
        {#each Object.keys(allowanceTypes) as type (type)}
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
      <DSCurrencyInput
        bind:amount={item.total}
        bind:currency={item.currency}
        items={$currencies.items}
        {errors}
        amountFieldName="total"
        currencyFieldName="currency"
        uiName="Total"
        disabledCurrency={currencySelectionDisabled}
        homeEquivalent={selectedCurrency && selectedCurrency.code !== "CAD"
          ? Number(item.total ?? 0) * Number(selectedCurrency.rate ?? 1)
          : null}
        rate={selectedCurrency?.rate}
        rateDate={selectedCurrency?.rate_date}
      />
      {#if showSettledTotalInput}
        <DSCurrencyInput
          bind:amount={item.settled_total}
          currency={homeCurrency?.id ?? ""}
          items={$currencies.items}
          {errors}
          amountFieldName="settled_total"
          currencyFieldName="settled_total_currency"
          uiName="Settled CAD Total"
          disabledCurrency={true}
          helperText="Enter the final CAD settlement amount before submitting."
        />
      {/if}
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
        >{isEditingAnotherUsersExpense ? "Back" : "Cancel"}</DsActionButton
      >
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
