<script lang="ts">
  import type { PageData } from "./$types";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import Icon from "@iconify/svelte";
  import { shortDate } from "$lib/utilities";
  import DsList from "$lib/components/DSList.svelte";
  export let data: PageData;
</script>

<div class="mx-auto space-y-4 p-4">
  <h1 class="text-2xl font-bold">Purchase Order Details</h1>

  <div class="space-y-2 rounded bg-neutral-100 p-4">
    <div class="flex items-center gap-2">
      <span class="font-semibold">PO Number:</span>
      <span>{data.po.po_number}</span>
      {#if data.po.status}
        <DsLabel
          color={data.po.status === "Active"
            ? "green"
            : data.po.status === "Closed"
              ? "gray"
              : data.po.status === "Cancelled"
                ? "orange"
                : "yellow"}
        >
          {data.po.status}
        </DsLabel>
      {/if}
      {#if data.po.type === "Cumulative"}
        <DsLabel color="teal"><Icon icon="mdi:sigma" width="24" class="inline-block" /></DsLabel>
      {/if}
    </div>

    {#if data.po.description}
      <div><span class="font-semibold">Description:</span> {data.po.description}</div>
    {/if}

    <div><span class="font-semibold">Date:</span> {shortDate(data.po.date)}</div>

    <div><span class="font-semibold">Total:</span> ${data.po.total}</div>

    {#if data.po.vendor}
      <div>
        <span class="font-semibold">Vendor:</span>
        <a href={`/vendors/${data.po.vendor}/details`} class="text-blue-600 hover:underline">
          {data.po.vendor_name}
          {#if data.po.vendor_alias}
            <span class="opacity-60">({data.po.vendor_alias})</span>
          {/if}
        </a>
      </div>
    {/if}

    {#if data.po.job}
      <div>
        <span class="font-semibold">Job:</span>
        <a href={`/jobs/${data.po.job}/details`} class="text-blue-600 hover:underline">
          {data.po.job_number}
        </a>
      </div>
    {/if}

    {#if data.po.division_code}
      <div>
        <span class="font-semibold">Division:</span>
        {data.po.division_code} — {data.po.division_name}
      </div>
    {/if}

    {#if data.po.payment_type}
      <div><span class="font-semibold">Payment Type:</span> {data.po.payment_type}</div>
    {/if}

    {#if data.po.type === "Recurring"}
      <div><span class="font-semibold">Frequency:</span> {data.po.frequency}</div>
      {#if data.po.end_date}
        <div><span class="font-semibold">End Date:</span> {shortDate(data.po.end_date)}</div>
      {/if}
    {/if}

    {#if data.po.approved}
      <div>
        <span class="font-semibold">Approved:</span>
        {shortDate(data.po.approved)} by {data.po.approver_name}
      </div>
    {/if}

    {#if data.po.second_approval}
      <div>
        <span class="font-semibold">Second Approval:</span>
        {shortDate(data.po.second_approval)} by {data.po.second_approver_name}
      </div>
    {/if}

    {#if data.po.rejected}
      <div>
        <span class="font-semibold">Rejected:</span>
        {shortDate(data.po.rejected)} — {data.po.rejection_reason}
      </div>
    {/if}
  </div>

  <div class="flex gap-2">
    <DsActionButton
      action={`/pos/${data.po.id}/edit`}
      icon="mdi:pencil"
      title="Edit Purchase Order"
      color="blue"
    />
  </div>

  <!-- Expenses referencing this PO -->
  <section class="mt-6 space-y-2">
    <h2 class="text-xl font-semibold">Expenses ({data.expenses.length})</h2>
    <DsList items={data.expenses} search={false}>
      {#snippet anchor(ex)}
        <a href={`/expenses/${ex.id}/details`} class="text-blue-600 hover:underline">
          {shortDate(ex.date)}
        </a>
      {/snippet}
      {#snippet headline(ex)}
        {ex.description || "Expense"}
      {/snippet}
      {#snippet byline({ total })}
        <span>${total}</span>
      {/snippet}
    </DsList>
  </section>
</div>
