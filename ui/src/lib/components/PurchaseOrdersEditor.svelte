<script lang="ts">
  import { flatpickrAction, fetchCategories, applyDefaultDivisionOnce } from "$lib/utilities";
  import { jobs } from "$lib/stores/jobs";
  import { vendors } from "$lib/stores/vendors";
  import { divisions } from "$lib/stores/divisions";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsFileSelect from "$lib/components/DsFileSelect.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { authStore } from "$lib/stores/auth";
  import { goto } from "$app/navigation";
  import type { PurchaseOrdersPageData } from "$lib/svelte-types";
  import type { CategoriesResponse, PoApproversResponse } from "$lib/pocketbase-types";
  import DsActionButton from "./DSActionButton.svelte";
  import DsLabel from "./DsLabel.svelte";
  import { untrack } from "svelte";

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
  let approvers = $state([] as PoApproversResponse[]);
  let secondApprovers = $state([] as PoApproversResponse[]);
  let showApproverField = $state(false);
  let showSecondApproverField = $state(false);

  // Watch for changes to the job and fetch categories accordingly
  $effect(() => {
    if (item.job) {
      fetchCategories(item.job).then((c) => (categories = c));
    }
  });

  // Default division from caller's profile if creating and empty
  $effect(() => applyDefaultDivisionOnce(item, data.editing));

  // Watch for changes to division, amount, or type to fetch approvers
  $effect(() => {
    if (item.division && item.total) {
      fetchApprovers();
    }
  });

  async function fetchApprovers() {
    try {
      // Build query parameters
      const queryParams = new URLSearchParams();
      if (isRecurring) {
        queryParams.append("type", "Recurring");
        queryParams.append("start_date", item.date || "");
        queryParams.append("end_date", item.end_date || "");
        queryParams.append("frequency", item.frequency || "");
      }

      // Fetch first approvers
      approvers = await pb.send(
        `/api/purchase_orders/approvers/${item.division}/${item.total}?${queryParams.toString()}`,
        {
          method: "GET",
        },
      );

      // Fetch second approvers
      secondApprovers = await pb.send(
        `/api/purchase_orders/second_approvers/${item.division}/${item.total}?${queryParams.toString()}`,
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
    item.uid = $authStore?.model?.id;

    // if the job is empty, set the category to empty
    if (item.job === "") {
      item.category = "";
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

<form
  class="flex w-full flex-col items-center gap-2 p-2"
  enctype="multipart/form-data"
  onsubmit={save}
>
  {#if isChildPO && data.parent_po_number}
    <span class="flex w-full gap-2 {errors.parent_po !== undefined ? 'bg-red-200' : ''}">
      <DsLabel color="cyan">Child PO of {data.parent_po_number}</DsLabel>
      {#if errors.parent_po !== undefined}
        <span class="text-red-600">{errors.parent_po.message}</span>
      {/if}
    </span>
  {/if}

  <DsSelector
    bind:value={item.type as string}
    items={[
      { id: "Normal", name: "Normal" },
      { id: "Cumulative", name: "Cumulative" },
      { id: "Recurring", name: "Recurring" },
    ]}
    {errors}
    fieldName="type"
    uiName="Purchase Order Type"
    disabled={isChildPO}
  >
    {#snippet optionTemplate(item)}
      {item.name}
    {/snippet}
  </DsSelector>

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

  <DsSelector
    bind:value={item.payment_type as string}
    items={[
      { id: "OnAccount", name: "On Account" },
      { id: "Expense", name: "Expense" },
      { id: "CorporateCreditCard", name: "Corporate Credit Card" },
    ]}
    {errors}
    fieldName="payment_type"
    uiName="Payment Type"
    disabled={isChildPO}
  >
    {#snippet optionTemplate(item)}
      {item.name}
    {/snippet}
  </DsSelector>

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
