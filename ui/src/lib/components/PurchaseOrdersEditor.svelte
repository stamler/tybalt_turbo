<script lang="ts">
  import flatpickr from "flatpickr";
  import { onMount } from "svelte";
  import { globalStore } from "$lib/stores/global";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import { authStore } from "$lib/stores/auth";
  import { goto } from "$app/navigation";
  import type { PurchaseOrdersPageData } from "$lib/svelte-types";

  let { data }: { data: PurchaseOrdersPageData } = $props();

  let calendarInput = $state<HTMLInputElement | null>(null);
  let endDateCalendarInput = $state<HTMLInputElement | null>(null);
  let errors = $state({} as any);
  let item = data.item;

  const isRecurring = $derived(item.type === "Recurring");

  async function save(event: Event) {
    event.preventDefault();
    item.uid = $authStore?.model?.id;

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

  // initialize flatpickr on the onMount lifecycle event
  onMount(() => {
    if (calendarInput) {
      flatpickr(calendarInput, {
        inline: true,
        minDate: "2024-06-01",
        // 2 months from now
        maxDate: new Date(new Date().setMonth(new Date().getMonth() + 3)),
        enableTime: false,
        dateFormat: "Y-m-d",
        defaultDate: new Date(),
      });
    }
  });

  // Use a reactive statement to initialize flatpickr for the end date input
  $effect(() => {
    if (isRecurring && endDateCalendarInput) {
      flatpickr(endDateCalendarInput, {
        inline: true,
        minDate: "2024-06-01",
        maxDate: new Date(new Date().setMonth(new Date().getMonth() + 3)),
        enableTime: false,
        dateFormat: "Y-m-d",
      });
    }
  });
</script>

<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css" />

<form
  class="flex w-full flex-col items-center gap-2 p-2"
  enctype="multipart/form-data"
  onsubmit={save}
>
  <DsSelector
    bind:value={item.type}
    items={[
      { id: "Normal", name: "Normal" },
      { id: "Cumulative", name: "Cumulative" },
      { id: "Recurring", name: "Recurring" },
    ]}
    {errors}
    fieldName="type"
    uiName="Purchase Order Type"
  >
    {#snippet optionTemplate(item)}
      {item.name}
    {/snippet}
  </DsSelector>

  <span class="flex w-full flex-col gap-2">
    <label for="date">Date</label>
    <input
      class="flex-1"
      type="text"
      name="date"
      placeholder="Date"
      bind:this={calendarInput}
      bind:value={item.date}
    />
  </span>

  {#if isRecurring}
    <span class="flex w-full flex-col gap-2">
      <label for="end_date">End Date</label>
      <input
        class="flex-1"
        type="text"
        name="end_date"
        placeholder="End Date"
        bind:this={endDateCalendarInput}
        bind:value={item.end_date}
      />
    </span>

    <DsSelector
      bind:value={item.frequency}
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

  <DsSelector
    bind:value={item.division}
    items={$globalStore.divisions}
    {errors}
    fieldName="division"
    uiName="Division"
  >
    {#snippet optionTemplate(item)}
      {item.code} - {item.name}
    {/snippet}
  </DsSelector>

  <DsTextInput
    bind:value={item.description}
    {errors}
    fieldName="description"
    uiName="Description"
  />

  <DsTextInput
    bind:value={item.total}
    {errors}
    fieldName="total"
    uiName="Total"
    type="number"
    step={0.01}
    min={0}
  />

  <DsSelector
    bind:value={item.payment_type}
    items={[
      { id: "OnAccount", name: "On Account" },
      { id: "Expense", name: "Expense" },
      { id: "CorporateCreditCard", name: "Corporate Credit Card" },
    ]}
    {errors}
    fieldName="payment_type"
    uiName="Payment Type"
  >
    {#snippet optionTemplate(item)}
      {item.name}
    {/snippet}
  </DsSelector>

  <DsTextInput
    bind:value={item.vendor_name}
    {errors}
    fieldName="vendor_name"
    uiName="Vendor Name"
  />

  <!-- File upload for attachment -->
  <span class="flex w-full flex-col gap-2">
    <label for="attachment">Attachment</label>
    <input
      id="attachment"
      type="file"
      onchange={(e) => (item.attachment = e.target?.files[0])}
      name="attachment"
    />
  </span>

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <button type="submit" class="rounded-sm bg-yellow-200 px-1 hover:bg-yellow-300">
        Save
      </button>
      <button type="button" onclick={() => goto("/purchase-orders/list")}>Cancel</button>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
