<script lang="ts">
  import type { PageData } from "./$types";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import Icon from "@iconify/svelte";
  import { shortDate } from "$lib/utilities";
  import DsList from "$lib/components/DSList.svelte";
  import { globalStore } from "$lib/stores/global";
  import { expensesEditingEnabled } from "$lib/stores/appConfig";
  import { pb } from "$lib/pocketbase";
  import { goto, invalidateAll } from "$app/navigation";
  let { data }: { data: PageData } = $props();
  const viewerId = pb.authStore.record?.id ?? "";
  let rejectModal: RejectModal;
  let showSecondApproverWhy = $state(false);
  const formatAmount = (value: number) => (Number.isFinite(value) ? value.toFixed(2) : String(value));
  const secondApproverMeta = $derived(data.secondApproverDiagnostics?.meta ?? null);
  const hasSecondApproverAlert = $derived(secondApproverMeta?.status === "required_no_candidates");
  const isRejected = $derived(data.po.status === "Unapproved" && data.po.rejected !== "");
  const displayStatus = $derived(isRejected ? "Rejected" : data.po.status);
  const isOwner = $derived(data.po.uid === viewerId);
  const canApproveOrReject = $derived(data.canApproveOrReject);

  async function refreshDetails() {
    await invalidateAll();
  }

  async function approvePo() {
    try {
      await pb.send(`/api/purchase_orders/${data.po.id}/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      await refreshDetails();
    } catch (e: any) {
      globalStore.addError(e?.response?.message || "Approve failed");
    }
  }

  function openRejectModal() {
    rejectModal?.openModal(data.po.id);
  }

  async function cancelPo() {
    try {
      await pb.send(`/api/purchase_orders/${data.po.id}/cancel`, { method: "POST" });
      goto("/pos/list");
    } catch (e: any) {
      globalStore.addError(e);
    }
  }

  async function closePo() {
    try {
      await pb.send(`/api/purchase_orders/${data.po.id}/close`, { method: "POST" });
      goto("/pos/list");
    } catch (e: any) {
      globalStore.addError(e);
    }
  }
</script>

<div class="mx-auto space-y-4 p-4">
  <h1 class="text-2xl font-bold">Purchase Order Details</h1>

  <div class="overflow-hidden rounded-sm">
    <div class="flex items-center justify-between bg-neutral-200 p-2">
      <span>{data.po.po_number === "" ? "no po number" : data.po.po_number}</span>
      <div class="flex items-center gap-2">
        {shortDate(data.po.date, true)}
        {#if hasSecondApproverAlert}
          <button
            type="button"
            class="rounded-sm border border-red-400 bg-red-100 px-2 py-0.5 text-red-700 hover:bg-red-200"
            title="Second approval ownership issue"
            onclick={() => {
              showSecondApproverWhy = !showSecondApproverWhy;
            }}
          >
            !
          </button>
        {/if}
        {#if data.po.status}
          <DsLabel
            color={displayStatus === "Active"
              ? "green"
              : displayStatus === "Closed"
                ? "gray"
                : displayStatus === "Cancelled"
                  ? "gray"
                  : displayStatus === "Rejected"
                    ? "red"
                    : "yellow"}
          >
            {displayStatus}
          </DsLabel>
        {/if}
      </div>
    </div>

    <div class="space-y-2 bg-neutral-100 p-4">
      {#if isRejected}
        <div class="rounded-sm border border-red-300 bg-red-50 p-3 text-sm text-red-900">
          <div class="font-semibold">This purchase order was rejected.</div>
          <div class="mt-1">
            {#if data.po.rejector_name}{data.po.rejector_name}{:else}An approver{/if}
            rejected it on {shortDate(data.po.rejected)}.
          </div>
          {#if data.po.rejection_reason}
            <div class="mt-1"><span class="font-semibold">Reason:</span> {data.po.rejection_reason}</div>
          {/if}
          {#if isOwner && $expensesEditingEnabled}
            <div class="mt-2">
              <DsActionButton
                action={`/pos/${data.po.id}/edit`}
                icon="mdi:pencil"
                title="Edit and Resubmit"
                color="red"
              />
            </div>
          {/if}
        </div>
      {/if}

      {#if hasSecondApproverAlert && secondApproverMeta}
        <div class="rounded-sm border border-red-300 bg-red-50 p-2 text-sm text-red-800">
          <div class="font-semibold">{secondApproverMeta.reason_message}</div>
          <div class="mt-1">
            <button
              type="button"
              class="underline hover:text-red-900"
              onclick={() => {
                showSecondApproverWhy = !showSecondApproverWhy;
              }}
            >
              {showSecondApproverWhy ? "Hide why" : "Why?"}
            </button>
          </div>
          {#if showSecondApproverWhy}
            <div class="mt-2 space-y-0.5 text-xs">
              <div>reason code: {secondApproverMeta.reason_code || "n/a"}</div>
              <div>evaluated amount: ${formatAmount(secondApproverMeta.evaluated_amount)}</div>
              <div>
                second-approval threshold:
                ${formatAmount(secondApproverMeta.second_approval_threshold)}
              </div>
              <div>second-stage timeout (hours): {secondApproverMeta.second_stage_timeout_hours}</div>
              <div>eligibility limit rule: {secondApproverMeta.limit_column || "n/a"}</div>
              <div>division: {data.po.division_code || data.po.division || "n/a"}</div>
              <div>kind: {data.po.kind || "n/a"}</div>
              <div>has job: {data.po.job ? "yes" : "no"}</div>
            </div>
          {/if}
        </div>
      {/if}

      {#if data.po.description}
        <div>{data.po.description}</div>
      {/if}

      <!-- Grouped Type, Total, and Approval Total -->
      <div class="flex flex-wrap gap-4">
        <div>
          <span class="font-semibold">Type:</span>
          {data.po.type}
          {#if data.po.type === "Recurring"}
            — {data.po.frequency}
            {#if data.po.end_date}
              until {shortDate(data.po.end_date, true)}
            {/if}
          {/if}
        </div>
        <div><span class="font-semibold">Total:</span> ${data.po.total}</div>
        {#if data.po.type === "Recurring"}
          <div><span class="font-semibold">Approval Total:</span> ${data.po.approval_total}</div>
        {/if}
      </div>

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

      {#if data.po.division_code}
        <div>
          <span class="font-semibold">Division:</span>
          {data.po.division_code} — {data.po.division_name}
        </div>
      {/if}

      {#if data.po.payment_type}
        <div><span class="font-semibold">Payment Type:</span> {data.po.payment_type}</div>
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

      {#if data.po.attachment}
        <div>
          <span class="font-semibold">Attachment:</span>
          <a
            href={`${PUBLIC_POCKETBASE_URL}/api/files/purchase_orders/${data.po.id}/${data.po.attachment}`}
            class="text-blue-600 hover:underline"
            target="_blank"
            rel="noopener noreferrer">Download</a
          >
        </div>
      {/if}

      {#if data.po.cancelled}
        <div>
          <span class="font-semibold">Cancelled:</span>
          {shortDate(data.po.cancelled)}{#if data.po.canceller}
            by {data.po.canceller}{/if}
        </div>
      {/if}

      {#if data.po.closed}
        <div>
          <span class="font-semibold">Closed:</span>
          {shortDate(data.po.closed)}{#if data.po.closer}
            by {data.po.closer}{/if}
        </div>
      {/if}

      {#if data.po.closed_by_system !== undefined}
        <div>
          <span class="font-semibold">Closed By System:</span>
          {data.po.closed_by_system ? "Yes" : "No"}
        </div>
      {/if}

      {#if data.po.job || data.po.client_name}
        <div class="flex flex-wrap items-center gap-4">
          {#if data.po.job}
            <div>
              <span class="font-semibold">Job:</span>
              <a href={`/jobs/${data.po.job}/details`} class="text-blue-600 hover:underline">
                {data.po.job_number}
              </a>
              {#if data.po.job_description}
                — {data.po.job_description}
              {/if}
            </div>
          {/if}

          {#if data.po.category_name}
            <DsLabel color="teal">{data.po.category_name}</DsLabel>
          {/if}

          {#if data.po.client_id}
            <div>
              <span class="font-semibold">Client:</span>
              <a
                href={`/clients/${data.po.client_id}/details`}
                class="text-blue-600 hover:underline"
              >
                {data.po.client_name}
              </a>
            </div>
          {:else if data.po.client_name}
            <div><span class="font-semibold">Client:</span> {data.po.client_name}</div>
          {/if}
        </div>
      {/if}

      {#if data.po.committed_expenses_count !== undefined}
        <div>
          <span class="font-semibold">Committed Expenses:</span>
          {data.po.committed_expenses_count}
        </div>
      {/if}

      {#if data.po.parent_po_number}
        <div>
          <span class="font-semibold">Parent PO:</span>
          <a href={`/pos/${data.po.parent_po}/details`} class="text-blue-600 hover:underline"
            >{data.po.parent_po_number}</a
          >
        </div>
      {/if}

      {#if data.po.priority_second_approver_name}
        <div>
          <span class="font-semibold">Priority Second Approver:</span>
          {data.po.priority_second_approver_name}
        </div>
      {/if}

      {#if data.po.rejector_name}
        <div><span class="font-semibold">Rejector:</span> {data.po.rejector_name}</div>
      {/if}

      {#if data.po.uid_name}
        <div><span class="font-semibold">Created By:</span> {data.po.uid_name}</div>
      {/if}
    </div>
  </div>
  {#if data.po.status === "Unapproved" && $expensesEditingEnabled}
    <div class="flex flex-wrap gap-2">
      {#if isOwner}
        <DsActionButton
          action={`/pos/${data.po.id}/edit`}
          icon="mdi:pencil"
          title={isRejected ? "Edit and Resubmit" : "Edit Purchase Order"}
          color="blue"
        />
      {/if}
      {#if canApproveOrReject && !isRejected}
        <DsActionButton action={() => approvePo()} icon="mdi:approve" title="Approve" color="green" />
        <DsActionButton
          action={() => openRejectModal()}
          icon="mdi:cancel"
          title="Reject"
          color="orange"
        />
      {/if}
    </div>
  {/if}

  {#if $globalStore.showAllUi || $globalStore.claims.includes("payables_admin")}
    <div class="mt-4 rounded-sm border border-neutral-300 bg-neutral-50 p-4">
      <h3 class="font-bold text-neutral-800">Admin Actions</h3>
      <div class="mt-2 flex gap-2">
        <DsActionButton
          action={cancelPo}
          icon="mdi:cancel"
          title="Cancel Purchase Order"
          color="red"
        />
        <DsActionButton
          action={closePo}
          icon="mdi:lock"
          title="Close Purchase Order"
          color="gray"
        />
      </div>
    </div>
  {/if}

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

  <RejectModal collectionName="purchase_orders" bind:this={rejectModal} on:refresh={refreshDetails} />
</div>
