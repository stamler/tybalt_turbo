<script lang="ts">
  import type { PageData } from "./$types";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import Icon from "@iconify/svelte";
  import { shortDate } from "$lib/utilities";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import { goto } from "$app/navigation";
  import { expensesEditingEnabled } from "$lib/stores/appConfig";

  export let data: PageData;
  const viewerId = pb.authStore.record?.id ?? "";
  let rejectModal: RejectModal;
  let expense = data.expense;
  let hasCommitAccess = false;
  let isOwner = false;
  let isApprover = false;

  $: hasCommitAccess = $globalStore.showAllUi || $globalStore.claims.includes("commit");
  $: isOwner = expense.uid === viewerId;
  $: isApprover = expense.approver === viewerId;

  async function refreshExpense() {
    try {
      expense = await pb.send(`/api/expenses/details/${expense.id}`, { method: "GET" });
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to refresh expense details");
    }
  }

  async function submitExpense() {
    try {
      await pb.send(`/api/expenses/${expense.id}/submit`, { method: "POST" });
      await refreshExpense();
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Submit failed");
    }
  }

  async function recallExpense() {
    try {
      await pb.send(`/api/expenses/${expense.id}/recall`, { method: "POST" });
      await refreshExpense();
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Recall failed");
    }
  }

  async function approveExpense() {
    try {
      await pb.send(`/api/expenses/${expense.id}/approve`, { method: "POST" });
      await refreshExpense();
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Approve failed");
    }
  }

  async function commitExpense() {
    try {
      await pb.send(`/api/expenses/${expense.id}/commit`, { method: "POST" });
      await refreshExpense();
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Commit failed");
    }
  }

  async function deleteExpense() {
    try {
      await pb.collection("expenses").delete(expense.id);
      goto("/expenses/list");
    } catch (error: any) {
      globalStore.addError(error?.response?.message || "Delete failed");
    }
  }

  function openRejectModal() {
    // @ts-ignore exported function on the component instance
    rejectModal?.openModal(expense.id);
  }

  function poDiffers(expenseVal: string, poVal: string): boolean {
    return !!expense.purchase_order && !!poVal && expenseVal !== poVal;
  }

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
      <span>{shortDate(expense.date)}</span>
      {#if expense.payment_type === "Mileage"}
        <DsLabel color="cyan">
          <Icon icon="mdi:road" width="24" class="inline-block" /> Mileage
        </DsLabel>
      {/if}
    </div>

    <div><span class="font-semibold">Submitted By:</span> {expense.uid_name}</div>

    <div>
      <span class="font-semibold">Description:</span> {expense.description}
      {#if poDiffers(expense.description, expense.po_description)}
        <span class="ml-1 inline-block rounded-sm border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs text-amber-600">PO: {expense.po_description}</span>
      {/if}
    </div>

    <div>
      <span class="font-semibold">Total:</span> ${expense.total}
      {#if expense.purchase_order && expense.po_total && expense.total !== expense.po_total}
        <span class="ml-1 inline-block rounded-sm border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs text-amber-600">PO: ${expense.po_total}</span>
      {/if}
    </div>

    {#if expense.payment_type}
      <div>
        <span class="font-semibold">Payment Type:</span>
        {paymentTypeLabel(expense.payment_type)}
        {#if poDiffers(expense.payment_type, expense.po_payment_type)}
          <span class="ml-1 inline-block rounded-sm border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs text-amber-600">PO: {paymentTypeLabel(expense.po_payment_type)}</span>
        {/if}
      </div>
    {/if}

    {#if expense.cc_last_4_digits}
      <div>
        <span class="font-semibold">Card:</span>
        ****{expense.cc_last_4_digits}
      </div>
    {/if}

    {#if expense.vendor}
      <div>
        <span class="font-semibold">Vendor:</span>
        <a href={`/vendors/${expense.vendor}/details`} class="text-blue-600 hover:underline">
          {expense.vendor_name}
          {#if expense.vendor_alias}
            <span class="opacity-60">({expense.vendor_alias})</span>
          {/if}
        </a>
        {#if poDiffers(expense.vendor, expense.po_vendor)}
          <span class="ml-1 inline-block rounded-sm border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs text-amber-600">PO: {expense.po_vendor_name}{#if expense.po_vendor_alias} ({expense.po_vendor_alias}){/if}</span>
        {/if}
      </div>
    {/if}

    {#if expense.purchase_order}
      <div>
        <span class="font-semibold">Purchase Order:</span>
        <a href={`/pos/${expense.purchase_order}/details`} class="text-blue-600 hover:underline">
          {expense.purchase_order_number}
        </a>
      </div>
    {/if}

    {#if expense.job}
      <div>
        <span class="font-semibold">Job:</span>
        <a href={`/jobs/${expense.job}/details`} class="text-blue-600 hover:underline">
          {expense.job_number}
          {#if expense.job_description}
            <span class="opacity-60">— {expense.job_description}</span>
          {/if}
        </a>
        {#if poDiffers(expense.job, expense.po_job)}
          <span class="ml-1 inline-block rounded-sm border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs text-amber-600">PO: {expense.po_job_number}{#if expense.po_job_description} — {expense.po_job_description}{/if}</span>
        {/if}
      </div>
    {/if}

    {#if expense.client_name}
      <div><span class="font-semibold">Client:</span> {expense.client_name}</div>
    {/if}

    {#if expense.division_code}
      <div>
        <span class="font-semibold">Division:</span>
        {expense.division_code} — {expense.division_name}
        {#if poDiffers(expense.division, expense.po_division)}
          <span class="ml-1 inline-block rounded-sm border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs text-amber-600">PO: {expense.po_division_code} — {expense.po_division_name}</span>
        {/if}
      </div>
    {/if}

    {#if expense.category_name}
      <div>
        <span class="font-semibold">Category:</span> {expense.category_name}
        {#if poDiffers(expense.category, expense.po_category)}
          <span class="ml-1 inline-block rounded-sm border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs text-amber-600">PO: {expense.po_category_name}</span>
        {/if}
      </div>
    {/if}

    {#if expense.kind_name}
      <div>
        <span class="font-semibold">Kind:</span> {expense.kind_name}
        {#if poDiffers(expense.kind, expense.po_kind)}
          <span class="ml-1 inline-block rounded-sm border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs text-amber-600">PO: {expense.po_kind_name}</span>
        {/if}
      </div>
    {/if}

    {#if expense.branch_name}
      <div>
        <span class="font-semibold">Branch:</span> {expense.branch_name}
        {#if poDiffers(expense.branch_name, expense.po_branch_name)}
          <span class="ml-1 inline-block rounded-sm border border-amber-300 bg-amber-100 px-2 py-0.5 text-xs text-amber-600">PO: {expense.po_branch_name}</span>
        {/if}
      </div>
    {/if}

    {#if expense.payment_type === "Mileage"}
      <div><span class="font-semibold">Distance:</span> {expense.distance} km</div>
    {/if}

    {#if expense.allowance_types && expense.allowance_types !== "[]"}
      <div>
        <span class="font-semibold">Allowance Types:</span>
        {expense.allowance_types}
      </div>
    {/if}

    {#if expense.attachment}
      <div class="flex items-center gap-2">
        <span class="font-semibold">Attachment:</span>
        <a
          href={`${PUBLIC_POCKETBASE_URL}/api/files/expenses/${expense.id}/${expense.attachment}`}
          target="_blank"
          class="inline-flex items-center gap-1 rounded-sm bg-blue-600 px-2 py-1 text-sm text-white hover:bg-blue-700"
        >
          <Icon icon="mdi:download" width="16" />
          Download
        </a>
        {#if expense.attachment_hash}
          <span class="font-mono text-sm opacity-70"
            >{expense.attachment_hash.slice(0, 8)}</span
          >
          <button
            type="button"
            class="text-neutral-500 hover:text-neutral-700"
            title="Copy full hash"
            on:click={() => navigator.clipboard.writeText(expense.attachment_hash)}
          >
            <Icon icon="mdi:content-copy" width="16" />
          </button>
        {/if}
      </div>
    {/if}

    {#if expense.pay_period_ending}
      <div>
        <span class="font-semibold">Pay Period Ending:</span>
        {shortDate(expense.pay_period_ending)}
      </div>
    {/if}

    <div>
      <span class="font-semibold">Submitted:</span>
      {expense.submitted ? "Yes" : "No"}
    </div>

    {#if expense.approved}
      <div>
        <span class="font-semibold">Approved:</span>
        {shortDate(expense.approved)} by {expense.approver_name}
      </div>
    {/if}

    {#if expense.rejected}
      <div class="rounded-sm border border-red-300 bg-red-50 p-3 space-y-1">
        <div class="flex items-center gap-2">
          <DsLabel color="red">
            <Icon icon="mdi:cancel" width="16" class="inline-block" /> Rejected
          </DsLabel>
          <span>{shortDate(expense.rejected)} by {expense.rejector_name}</span>
        </div>
        {#if expense.rejection_reason}
          <div class="text-red-700">{expense.rejection_reason}</div>
        {/if}
      </div>
    {/if}

    {#if expense.committed}
      <div>
        <span class="font-semibold">Committed:</span>
        {shortDate(expense.committed)}
        {#if expense.committed_week_ending}
          (Week ending {shortDate(expense.committed_week_ending)})
        {/if}
      </div>
    {/if}
  </div>

  {#if $expensesEditingEnabled}
    <div class="flex flex-wrap gap-2">
      {#if isOwner && !expense.submitted}
        <DsActionButton action={`/expenses/${expense.id}/edit`} icon="mdi:pencil" title="Edit" color="blue" />
        <DsActionButton action={() => submitExpense()} icon="mdi:send" title="Submit" color="blue" />
      {/if}

      {#if isOwner && ((expense.submitted && expense.approved === "") || expense.rejected !== "") && expense.committed === ""}
        <DsActionButton action={() => recallExpense()} icon="mdi:rewind" title="Recall" color="orange" />
      {/if}

      {#if isOwner && !expense.submitted && expense.committed === ""}
        <DsActionButton action={() => deleteExpense()} icon="mdi:delete" title="Delete" color="red" />
      {/if}

      {#if isApprover && expense.submitted && expense.approved === "" && expense.rejected === "" && expense.committed === ""}
        <DsActionButton action={() => approveExpense()} icon="mdi:approve" title="Approve" color="green" />
      {/if}

      {#if (isApprover || hasCommitAccess) && expense.submitted && expense.rejected === "" && expense.committed === ""}
        <DsActionButton action={() => openRejectModal()} icon="mdi:cancel" title="Reject" color="orange" />
      {/if}

      {#if hasCommitAccess && expense.submitted && expense.approved !== "" && expense.rejected === "" && expense.committed === ""}
        <DsActionButton action={() => commitExpense()} icon="mdi:check-all" title="Commit" color="green" />
      {/if}
    </div>
  {/if}

  <RejectModal collectionName="expenses" bind:this={rejectModal} on:refresh={refreshExpense} />
</div>
