<script lang="ts">
  import { flatpickrAction, fetchCategories } from "$lib/utilities";
  import { globalStore } from "$lib/stores/global";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsFileSelect from "$lib/components/DsFileSelect.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { authStore } from "$lib/stores/auth";
  import { goto } from "$app/navigation";
  import type { ExpensesPageData } from "$lib/svelte-types";
  import type { CategoriesResponse, ExpensesAllowanceTypesOptions } from "$lib/pocketbase-types";
  import { isExpensesResponse } from "$lib/pocketbase-types";
  import DsActionButton from "./DSActionButton.svelte";
  import CumulativePOOverflowModal from "./CumulativePOOverflowModal.svelte";

  let { data }: { data: ExpensesPageData } = $props();

  let errors = $state({} as any);
  let item = $state(data.item);
  let overflowModal: CumulativePOOverflowModal;

  let categories = $state([] as CategoriesResponse[]);

  // create a local state object to hold the allowance types
  const allowanceTypes = $state({
    Breakfast: false,
    Lunch: false,
    Dinner: false,
    Lodging: false,
  });

  // Watch for changes to the job and fetch categories accordingly
  $effect(() => {
    if (item.job) {
      fetchCategories(item.job).then((c) => (categories = c));
    }
  });

  async function save(event: Event) {
    event.preventDefault();
    item.uid = $authStore?.model?.id;

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

  function handleCreateChild(event: CustomEvent) {
    const { parent_po, overflow_amount } = event.detail;
    // TODO: Navigate to create child PO page with pre-filled data
    console.log("Create child PO", parent_po, overflow_amount);
  }
</script>

<svelte:head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css" />
</svelte:head>

<CumulativePOOverflowModal bind:this={overflowModal} on:createChild={handleCreateChild} />

<form
  class="flex w-full flex-col items-center gap-2 p-2"
  enctype="multipart/form-data"
  onsubmit={save}
>
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

  {#if isExpensesResponse(item) && item.purchase_order !== ""}
    <span class="flex w-full gap-2">
      <DsLabel color="cyan">PO {item.expand.purchase_order.po_number}</DsLabel>
    </span>
  {/if}

  <DsSelector
    bind:value={item.division as string}
    items={$globalStore.divisions}
    {errors}
    fieldName="division"
    uiName="Division"
    disabled={item.purchase_order !== ""}
  >
    {#snippet optionTemplate(item)}
      {item.code} - {item.name}
    {/snippet}
  </DsSelector>

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
    disabled={item.purchase_order !== ""}
  >
    {#snippet optionTemplate(item)}
      {item.name}
    {/snippet}
  </DsSelector>

  <!-- Job + category fields are only allowed if a purchase order is present or
   if the payment type is Allowance, Mileage, FuelCard, or
   PersonalReimbursement, so we should 
     1. show them as uneditable fields when creating an expense from an existing
        purchase order (purchase_order !== "")
     2. show them as editable if the payment type is Allowance, Mileage,
        FuelCard, or PersonalReimbursement
     3. hide them otherwise
   -->
  {#if $globalStore.jobsIndex !== null}
    <DsAutoComplete
      bind:value={item.job as string}
      index={$globalStore.jobsIndex}
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
      disabled={item.purchase_order !== ""}
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
      disabled={item.purchase_order !== ""}
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
      {#if $globalStore.vendorsIndex !== null}
        <DsAutoComplete
          bind:value={item.vendor as string}
          index={$globalStore.vendorsIndex}
          {errors}
          fieldName="vendor"
          uiName="Vendor"
        >
          {#snippet resultTemplate(item)}{item.name} ({item.alias}){/snippet}
        </DsAutoComplete>
      {/if}
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
      <DsActionButton type="submit">Save</DsActionButton>
      <DsActionButton action="/expenses/list">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
