<script lang="ts">
  import type { PageData } from "./$types";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import Icon from "@iconify/svelte";
  import { shortDate } from "$lib/utilities";

  export let data: PageData;

  function paymentTypeLabel(pt: string) {
    switch (pt) {
      case "CorporateCreditCard":
        return "Corporate Credit Card";
      case "OnAccount":
        return "On Account";
      case "PersonalReimbursement":
        return "Personal Reimbursement";
      default:
        return pt;
    }
  }
</script>

<div class="mx-auto space-y-4 p-4">
  <h1 class="text-2xl font-bold">Expense Details</h1>

  <div class="space-y-2 rounded bg-neutral-100 p-4">
    <div class="flex items-center gap-2">
      <span class="font-semibold">Date:</span>
      <span>{shortDate(data.expense.date)}</span>
      {#if data.expense.payment_type === "Mileage"}
        <DsLabel color="cyan">
          <Icon icon="mdi:road" width="24" class="inline-block" /> Mileage
        </DsLabel>
      {/if}
    </div>

    <div><span class="font-semibold">Description:</span> {data.expense.description}</div>

    <div><span class="font-semibold">Total:</span> ${data.expense.total}</div>

    {#if data.expense.payment_type}
      <div>
        <span class="font-semibold">Payment Type:</span>
        {paymentTypeLabel(data.expense.payment_type)}
      </div>
    {/if}

    {#if data.expense.vendor}
      <div>
        <span class="font-semibold">Vendor:</span>
        <a href={`/vendors/${data.expense.vendor}/details`} class="text-blue-600 hover:underline">
          {data.expense.vendor_name}
          {#if data.expense.vendor_alias}
            <span class="opacity-60">({data.expense.vendor_alias})</span>
          {/if}
        </a>
      </div>
    {/if}

    {#if data.expense.purchase_order}
      <div>
        <span class="font-semibold">Purchase Order:</span>
        <a
          href={`/pos/${data.expense.purchase_order}/details`}
          class="text-blue-600 hover:underline"
        >
          {data.expense.purchase_order_number}
        </a>
      </div>
    {/if}

    {#if data.expense.job}
      <div>
        <span class="font-semibold">Job:</span>
        <a href={`/jobs/${data.expense.job}/details`} class="text-blue-600 hover:underline">
          {data.expense.job_number}
        </a>
      </div>
    {/if}

    {#if data.expense.division_code}
      <div>
        <span class="font-semibold">Division:</span>
        {data.expense.division_code} — {data.expense.division_name}
      </div>
    {/if}

    {#if data.expense.payment_type === "Mileage"}
      <div><span class="font-semibold">Distance:</span> {data.expense.distance} km</div>
    {/if}

    {#if data.expense.approved}
      <div>
        <span class="font-semibold">Approved:</span>
        {shortDate(data.expense.approved)} by {data.expense.approver_name}
      </div>
    {/if}

    {#if data.expense.rejected}
      <div>
        <span class="font-semibold">Rejected:</span>
        {shortDate(data.expense.rejected)} — {data.expense.rejection_reason}
      </div>
    {/if}
  </div>

  <div class="flex gap-2">
    <DsActionButton
      action={`/expenses/${data.expense.id}/edit`}
      icon="mdi:pencil"
      title="Edit Expense"
      color="blue"
    />
  </div>
</div>
