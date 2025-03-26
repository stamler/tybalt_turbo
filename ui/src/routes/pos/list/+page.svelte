<script lang="ts">
  import Icon from "@iconify/svelte";
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import { pb } from "$lib/pocketbase";
  import { type UnsubscribeFunc } from "pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import type { PurchaseOrdersResponse } from "$lib/pocketbase-types";
  import { authStore } from "$lib/stores/auth";
  import { globalStore } from "$lib/stores/global";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import { shortDate } from "$lib/utilities";
  import { onMount, onDestroy } from "svelte";
  // import { toastStore, type ToastSettings } from "@skeletonlabs/skeleton";

  let rejectModal: RejectModal;

  // Load the initial data
  let { data }: { data: PageData } = $props();
  let items = $state(data.items);
  let unsubscribeFunc: UnsubscribeFunc;

  // create a map of purchase order id to augmented data
  let augmentedMap = $state(new Map(data.augmentedItems?.map((item) => [item.id, item]) ?? []));

  // Set to true to show all buttons regardless of user permissions or PO
  // status. This is used for testing purposes.
  const deactivateButtonHiding = $state(false);

  function poMayBeApprovedOrRejectedByUser(po: PurchaseOrdersResponse): boolean {
    if (deactivateButtonHiding) return true;
    if (po.status === "Unapproved") {
      // po is unapproved
      if (po.rejected === "") {
        // po is not rejected
        if ($globalStore.user_po_permission_data.claims.includes("po_approver")) {
          // user has po_approver claim
          if (
            po.approval_total <= $globalStore.user_po_permission_data.max_amount ||
            (po.approved === "" && po.approver === $authStore?.model?.id)
          ) {
            // po is below the user's max amount or the user is the specified
            // approver and the PO has not yet been approved
            if (
              $globalStore.user_po_permission_data.divisions.includes(po.division) ||
              $globalStore.user_po_permission_data.divisions.length === 0
            ) {
              // po is in the user's division or the user has no divisions
              if (po.approval_total >= $globalStore.user_po_permission_data.lower_threshold) {
                // po is above the user's lower threshold (in the same tier as the user's max amount)
                // user can approve
                return true;
              }
            }
          }
        }
      }
    }
    return false;
  }

  function poMayBeCancelledByUser(po: PurchaseOrdersResponse): boolean {
    if (deactivateButtonHiding) return true;
    if (po.status === "Active") {
      if ($globalStore.user_po_permission_data.claims.includes("payables_admin")) {
        // user has payables_admin claim
        const augmented = augmentedMap.get(po.id);
        if (augmented !== undefined) {
          // return true if there are no expenses associated with the PO
          return augmented.committed_expenses_count === 0;
        }
      }
    }
    return false;
  }

  function poMayBeClosedByUser(po: PurchaseOrdersResponse): boolean {
    if (deactivateButtonHiding) return true;
    if (po.status === "Active") {
      if (po.type !== "Normal") {
        // only normal POs can be closed manually
        if ($globalStore.user_po_permission_data.claims.includes("payables_admin")) {
          // user has payables_admin claim
          const augmented = augmentedMap.get(po.id);
          if (augmented !== undefined) {
            // return true if there is at least one committed expense associated with the PO
            return augmented.committed_expenses_count > 0;
          }
        }
      }
    }
    return false;
  }

  onMount(async () => {
    // Subscribe to the purchase_orders collection and act on the changes
    unsubscribeFunc = await pb.collection("purchase_orders").subscribe<PurchaseOrdersResponse>(
      "*",
      (e) => {
        // return immediately if items is not an array
        if (!Array.isArray(items)) return;
        switch (e.action) {
          case "create":
            // Insert the new item at the top of the list
            items = [e.record, ...items];
            break;
          case "update":
            items = items.map((item) => (item.id === e.record.id ? e.record : item));
            break;
          case "delete":
            items = items.filter((item) => item.id !== e.record.id);
            break;
        }
      },
      {
        expand:
          "uid.profiles_via_uid,approver.profiles_via_uid,division,vendor,job,job.client,rejector.profiles_via_uid,category,second_approver.profiles_via_uid,second_approver_claim,parent_po,priority_second_approver.profiles_via_uid",
        sort: "-date",
      },
    );
  });

  onDestroy(async () => {
    unsubscribeFunc();
  });

  async function del(id: string): Promise<void> {
    // return immediately if items is not an array
    if (!Array.isArray(items)) return;

    try {
      await pb.collection("purchase_orders").delete(id);
    } catch (error: any) {
      globalStore.addError(error?.response?.message);
    }
  }

  async function approve(id: string): Promise<void> {
    try {
      await pb.send(`/api/purchase_orders/${id}/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
    } catch (error: any) {
      globalStore.addError(error?.response?.message);
    }
  }

  async function cancel(id: string): Promise<void> {
    try {
      await pb.send(`/api/purchase_orders/${id}/cancel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
    } catch (error: any) {
      globalStore.addError(error?.response?.message);
    }
  }

  function openRejectModal(poId: string) {
    rejectModal?.openModal(poId);
  }

  async function closePurchaseOrder(id: string) {
    try {
      await pb.send(`/api/purchase_orders/${id}/close`, { method: "POST" });

      // Show success toast
      /*
      const t: ToastSettings = {
        message: "Purchase order closed successfully",
        background: "variant-filled-success",
      };
      toastStore.trigger(t);
      */
    } catch (error: any) {
      globalStore.addError(error?.response?.message);

      // Show error toast
      /*  
      const t: ToastSettings = {
        message: error.message || "Error closing purchase order",
        background: "variant-filled-error",
      };
      toastStore.trigger(t);
      */
    }
  }
</script>

{#snippet anchor(item: PurchaseOrdersResponse)}
  <span class="flex flex-col items-center gap-2">
    {#if item.status === "Active"}
      <DsLabel style="inverted" color="green">
        {item.po_number}
      </DsLabel>
    {:else if item.status === "Unapproved"}
      {item.date}
    {:else}
      {item.po_number}
    {/if}
    <!-- <DsActionButton
      action={() => navigator.clipboard.writeText(JSON.stringify(item))}
      icon="mdi:clipboard-outline"
      title="Copy"
      color="blue"
    /> -->
    {#if item.type === "Cumulative"}
      <DsLabel color="teal">
        <!-- <Icon icon="mdi:chart-bell-curve-cumulative" width="24px" class="inline-block" /> -->
        <Icon icon="mdi:sigma" width="24px" class="inline-block" />
      </DsLabel>
    {/if}
  </span>
{/snippet}

{#snippet headline({ total, payment_type, parent_po, expand }: PurchaseOrdersResponse)}
  <span class="flex items-center gap-2">
    ${total}
    {payment_type}
    <span class="flex items-center gap-0">
      <Icon icon="mdi:store" width="24px" class="inline-block" />
      {expand?.vendor.name} ({expand?.vendor.alias})
    </span>
    {#if parent_po !== ""}
      <DsLabel color="blue">
        child of {expand?.parent_po?.po_number}
      </DsLabel>
    {/if}
  </span>
{/snippet}

{#snippet byline({
  cancelled,
  canceller,
  description,
  rejected,
  expand,
  rejection_reason,
  status,
}: PurchaseOrdersResponse)}
  <span class="flex items-center gap-2">
    {description}
    {#if rejected !== ""}
      <DsLabel color="red" title={`Rejected ${shortDate(rejected)}: ${rejection_reason}`}>
        <Icon icon="mdi:cancel" width="24px" class="inline-block" />
        {expand?.rejector.expand?.profiles_via_uid.given_name}
        {expand?.rejector.expand?.profiles_via_uid.surname}
      </DsLabel>
    {:else if status === "Cancelled"}
      <DsLabel color="orange" title={`Cancelled ${shortDate(cancelled)}`}>
        <Icon icon="mdi:cancel" width="24px" class="inline-block" />
      </DsLabel>
    {/if}
  </span>
{/snippet}

{#snippet line1(item: PurchaseOrdersResponse)}
  <span>
    {item.expand?.uid.expand?.profiles_via_uid.given_name}
    {item.expand?.uid.expand?.profiles_via_uid.surname}
    {#if item.status !== "Unapproved"}
      ({shortDate(item.date)})
    {/if}
    / {item.expand?.division.code}
    {item.expand?.division.name}
  </span>
{/snippet}

{#snippet line2(item: PurchaseOrdersResponse)}
  {#if item.job !== ""}
    <span class="flex items-center gap-1">
      {item.expand.job.number} - {item.expand.job.expand.client.name}:
      {item.expand.job.description}
      {#if item.expand?.category !== undefined}
        <DsLabel color="teal">{item.expand.category.name}</DsLabel>
      {/if}
    </span>
  {/if}
{/snippet}
{#snippet line3(item: PurchaseOrdersResponse)}
  <span class="flex items-center gap-1">
    <!-- if the item is recurring, show the frequency -->
    {#if item.type === "Recurring"}
      <DsLabel color="cyan">
        <!-- Perhaps use Japanese character for monthly payment-->
        <Icon icon="mdi:recurring-payment" width="24px" class="inline-block" />
        {item.frequency} until {shortDate(item.end_date, true)}
      </DsLabel>
    {/if}
    {#if item.approved !== ""}
      <Icon icon="material-symbols:order-approve-outline" width="24px" class="inline-block" />
      {item.expand.approver.expand?.profiles_via_uid.given_name}
      {item.expand.approver.expand?.profiles_via_uid.surname}
      ({shortDate(item.approved)})
    {:else}
      <Icon icon="mdi:timer-sand" width="24px" class="inline-block" />
      {item.expand.approver.expand?.profiles_via_uid.given_name}
      {item.expand.approver.expand?.profiles_via_uid.surname}
    {/if}
    {#if item.second_approver !== "" && item.second_approval !== ""}
      <!-- Item has second approval -->
      <span class="flex items-center gap-1">
        /
        <Icon icon="material-symbols:order-approve-outline" width="24px" class="inline-block" />
        {item.expand.second_approver.expand?.profiles_via_uid.given_name}
        {item.expand.second_approver.expand?.profiles_via_uid.surname}
        ({shortDate(item.second_approval)})
      </span>
    {/if}
    {#if item.status === "Unapproved" && item.second_approval === "" && item.approved !== ""}
      <!-- Approved, but second approval is required -->
      <span class="flex items-center gap-1">
        /
        <Icon icon="mdi:timer-sand" width="24px" class="inline-block" />
        {item.expand.priority_second_approver?.expand.profiles_via_uid.given_name}
        {item.expand.priority_second_approver?.expand.profiles_via_uid.surname}
      </span>
    {/if}
    {#if item.attachment}
      <a
        href={`${PUBLIC_POCKETBASE_URL}/api/files/${item.collectionId}/${item.id}/${item.attachment}`}
        target="_blank"
      >
        <DsFileLink filename={item.attachment as string} />
      </a>
    {/if}
  </span>
{/snippet}

{#snippet actions(item: PurchaseOrdersResponse)}
  {#if item.status === "Active"}
    <DsActionButton
      action={`/expenses/add/${item.id}`}
      icon="mdi:add-bold"
      title="Create Expense"
      color="green"
    />
    {#if poMayBeCancelledByUser(item)}
      <DsActionButton
        action={() => cancel(item.id)}
        icon="mdi:cancel"
        title="Cancel"
        color="orange"
      />
    {/if}
    {#if poMayBeClosedByUser(item)}
      <DsActionButton
        action={() => closePurchaseOrder(item.id)}
        icon="mdi:curtains-closed"
        title="Close Purchase Order"
        color="orange"
      />
    {/if}
  {/if}
  {#if item.status === "Unapproved"}
    <DsActionButton
      action={`/pos/${item.id}/edit`}
      icon="mdi:edit-outline"
      title="Edit"
      color="blue"
    />
    {#if poMayBeApprovedOrRejectedByUser(item)}
      <DsActionButton
        action={() => approve(item.id)}
        icon="mdi:approve"
        title="Approve"
        color="green"
      />
      <DsActionButton
        action={() => openRejectModal(item.id)}
        icon="mdi:cancel"
        title="Reject"
        color="orange"
      />
    {/if}
    <DsActionButton action={() => del(item.id)} icon="mdi:delete" title="Delete" color="red" />
  {/if}
{/snippet}

<RejectModal collectionName="purchase_orders" bind:this={rejectModal} />
<DsList
  items={items as PurchaseOrdersResponse[]}
  search={true}
  inListHeader="Purchase Orders"
  {anchor}
  {headline}
  {byline}
  {line1}
  {line2}
  {line3}
  {actions}
/>
