<script lang="ts">
  import type { PageData } from "./$types";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import Icon from "@iconify/svelte";
  import { shortDate } from "$lib/utilities";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";

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

  <div class="space-y-2 rounded-sm bg-neutral-100 p-4">
    <div class="flex items-center gap-2">
      <span class="font-semibold">Date:</span>
      <span>{shortDate(data.expense.date)}</span>
      {#if data.expense.payment_type === "Mileage"}
        <DsLabel color="cyan">
          <Icon icon="mdi:road" width="24" class="inline-block" /> Mileage
        </DsLabel>
      {/if}
    </div>

    <div><span class="font-semibold">Submitted By:</span> {data.expense.uid_name}</div>

    <div><span class="font-semibold">Description:</span> {data.expense.description}</div>

    <div><span class="font-semibold">Total:</span> ${data.expense.total}</div>

    {#if data.expense.payment_type}
      <div>
        <span class="font-semibold">Payment Type:</span>
        {paymentTypeLabel(data.expense.payment_type)}
      </div>
    {/if}

    {#if data.expense.cc_last_4_digits}
      <div>
        <span class="font-semibold">Card:</span>
        ****{data.expense.cc_last_4_digits}
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
          {#if data.expense.job_description}
            <span class="opacity-60">— {data.expense.job_description}</span>
          {/if}
        </a>
      </div>
    {/if}

    {#if data.expense.client_name}
      <div><span class="font-semibold">Client:</span> {data.expense.client_name}</div>
    {/if}

    {#if data.expense.division_code}
      <div>
        <span class="font-semibold">Division:</span>
        {data.expense.division_code} — {data.expense.division_name}
      </div>
    {/if}

    {#if data.expense.category_name}
      <div><span class="font-semibold">Category:</span> {data.expense.category_name}</div>
    {/if}

    {#if data.expense.branch_name}
      <div><span class="font-semibold">Branch:</span> {data.expense.branch_name}</div>
    {/if}

    {#if data.expense.payment_type === "Mileage"}
      <div><span class="font-semibold">Distance:</span> {data.expense.distance} km</div>
    {/if}

    {#if data.expense.allowance_types && data.expense.allowance_types !== "[]"}
      <div>
        <span class="font-semibold">Allowance Types:</span>
        {data.expense.allowance_types}
      </div>
    {/if}

    {#if data.expense.attachment}
      <div class="flex items-center gap-2">
        <span class="font-semibold">Attachment:</span>
        <a
          href={`${PUBLIC_POCKETBASE_URL}/api/files/expenses/${data.expense.id}/${data.expense.attachment}`}
          target="_blank"
          class="inline-flex items-center gap-1 rounded-sm bg-blue-600 px-2 py-1 text-sm text-white hover:bg-blue-700"
        >
          <Icon icon="mdi:download" width="16" />
          Download
        </a>
        {#if data.expense.attachment_hash}
          <span class="font-mono text-sm opacity-70"
            >{data.expense.attachment_hash.slice(0, 8)}</span
          >
          <button
            type="button"
            class="text-neutral-500 hover:text-neutral-700"
            title="Copy full hash"
            on:click={() => navigator.clipboard.writeText(data.expense.attachment_hash)}
          >
            <Icon icon="mdi:content-copy" width="16" />
          </button>
        {/if}
      </div>
    {/if}

    {#if data.expense.pay_period_ending}
      <div>
        <span class="font-semibold">Pay Period Ending:</span>
        {shortDate(data.expense.pay_period_ending)}
      </div>
    {/if}

    <div>
      <span class="font-semibold">Submitted:</span>
      {data.expense.submitted ? "Yes" : "No"}
    </div>

    {#if data.expense.approved}
      <div>
        <span class="font-semibold">Approved:</span>
        {shortDate(data.expense.approved)} by {data.expense.approver_name}
      </div>
    {/if}

    {#if data.expense.rejected}
      <div>
        <span class="font-semibold">Rejected:</span>
        {shortDate(data.expense.rejected)} by {data.expense.rejector_name}
      </div>
      {#if data.expense.rejection_reason}
        <div>
          <span class="font-semibold">Rejection Reason:</span>
          {data.expense.rejection_reason}
        </div>
      {/if}
    {/if}

    {#if data.expense.committed}
      <div>
        <span class="font-semibold">Committed:</span>
        {shortDate(data.expense.committed)}
        {#if data.expense.committed_week_ending}
          (Week ending {shortDate(data.expense.committed_week_ending)})
        {/if}
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
