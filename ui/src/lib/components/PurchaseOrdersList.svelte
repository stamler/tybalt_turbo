<script lang="ts">
  import Icon from "@iconify/svelte";
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import { pb } from "$lib/pocketbase";
  import { type UnsubscribeFunc } from "pocketbase";
  import { augmentedProxySubscription } from "$lib/utilities";
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import type {
    PurchaseOrdersAugmentedResponse,
    PurchaseOrdersResponse,
  } from "$lib/pocketbase-types";
  import { authStore } from "$lib/stores/auth";
  import { globalStore } from "$lib/stores/global";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import { shortDate } from "$lib/utilities";
  import { onMount, onDestroy } from "svelte";
  // import { toastStore, type ToastSettings } from "@skeletonlabs/skeleton";

  let rejectModal: RejectModal;

  // Load the initial data
  let { inListHeader, data }: { inListHeader?: string; data: PageData } = $props();
  let items = $state(data.items);

  // Subscribe to the base collection but update the items from the augmented
  // view
  let unsubscribeFunc: UnsubscribeFunc;
  onMount(async () => {
    if (items === undefined) {
      return;
    }
    unsubscribeFunc = await augmentedProxySubscription<
      PurchaseOrdersResponse,
      PurchaseOrdersAugmentedResponse
    >(items, "purchase_orders", "purchase_orders_augmented", (newItems) => {
      items = newItems;
    });
  });
  onDestroy(async () => {
    unsubscribeFunc();
  });

  // Set to true to show all buttons regardless of user permissions or PO
  // status. This is used for testing purposes.
  const deactivateButtonHiding = $state(false);

  function poMayBeApprovedOrRejectedByUser(po: PurchaseOrdersAugmentedResponse): boolean {
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
              if (
                po.approval_total >= $globalStore.user_po_permission_data.lower_threshold ||
                po.uid === $authStore?.model?.id
              ) {
                // po is above the user's lower threshold (in the same tier as
                // the user's max amount) or the user created the PO (it's
                // annoying to have to go through the approval process if you
                // created it and you're qualified to approve it)
                return true;
              }
            }
          }
        }
      }
    }
    return false;
  }

  function poMayBeCancelledByUser(po: PurchaseOrdersAugmentedResponse): boolean {
    if (deactivateButtonHiding) return true;
    if (po.status === "Active") {
      if ($globalStore.user_po_permission_data.claims.includes("payables_admin")) {
        // user has payables_admin claim
        return po.committed_expenses_count === 0;
      }
    }
    return false;
  }

  function poMayBeClosedByUser(po: PurchaseOrdersAugmentedResponse): boolean {
    if (deactivateButtonHiding) return true;
    if (po.status === "Active") {
      if (po.type !== "Normal") {
        // only normal POs can be closed manually
        if ($globalStore.user_po_permission_data.claims.includes("payables_admin")) {
          // user has payables_admin claim
          // return true if there is at least one committed expense associated with the PO
          return po.committed_expenses_count > 0;
        }
      }
    }
    return false;
  }

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

<RejectModal collectionName="purchase_orders" bind:this={rejectModal} />
<DsList items={items as PurchaseOrdersAugmentedResponse[]} search={true} {inListHeader}>
  {#snippet anchor(item: PurchaseOrdersAugmentedResponse)}
    <span class="flex flex-col items-center gap-2">
      {#if item.status === "Active"}
        <DsLabel style="inverted" color="green">
          <a href={`/pos/${item.id}/details`} class="hover:underline">
            {item.po_number}
          </a>
        </DsLabel>
      {:else if item.status === "Unapproved"}
        <a href={`/pos/${item.id}/details`} class="text-blue-600 hover:underline">
          {item.date}
        </a>
      {:else}
        <a href={`/pos/${item.id}/details`} class="text-blue-600 hover:underline">
          {item.po_number}
        </a>
      {/if}
    </span>
  {/snippet}

  {#snippet headline({ vendor_name, vendor_alias }: PurchaseOrdersAugmentedResponse)}
    <span class="flex items-center gap-2">
      {vendor_name}
      {#if vendor_alias}
        ({vendor_alias})
      {/if}
    </span>
  {/snippet}

  {#snippet byline({ total, status }: PurchaseOrdersAugmentedResponse)}
    <span class="flex items-center gap-2">
      ${total}
      {#if status !== "Active"}
        <DsLabel color={status === "Closed" || status === "Cancelled" ? "gray" : "yellow"}>
          {status}
        </DsLabel>
      {/if}
    </span>
  {/snippet}

  {#snippet line1({ description }: PurchaseOrdersAugmentedResponse)}
    {description}
  {/snippet}

  {#snippet line2({
    job_number,
    client_name,
    job_description,
    category_name,
  }: PurchaseOrdersAugmentedResponse)}
    {#if job_number !== ""}
      <span class="flex items-center gap-1">
        {job_number} - {client_name}:
        {job_description}
        {#if category_name !== ""}
          <DsLabel color="teal">{category_name}</DsLabel>
        {/if}
      </span>
    {/if}
  {/snippet}
  {#snippet line3(item: PurchaseOrdersAugmentedResponse)}
    <span class="flex items-center gap-1">
      <!-- if the item is cumulative, show the sigma icon -->
      {#if item.type === "Cumulative"}
        <DsLabel color="cyan">
          <!-- <Icon icon="mdi:chart-bell-curve-cumulative" width="24px" class="inline-block" /> -->
          <Icon icon="mdi:sigma" width="24px" class="inline-block" />
        </DsLabel>
      {/if}
      <!-- if the item is recurring, show the frequency -->
      {#if item.type === "Recurring"}
        <DsLabel color="cyan">
          <!-- Perhaps use Japanese character for monthly payment-->
          <Icon icon="mdi:recurring-payment" width="24px" class="inline-block" />
          {item.frequency} until {shortDate(item.end_date, true)}
        </DsLabel>
      {/if}
      {#if item.status === "Unapproved" && item.second_approval === "" && item.approved !== ""}
        <!-- Approved, but second approval is required -->
        <span class="flex items-center gap-1">
          /
          <Icon icon="mdi:timer-sand" width="24px" class="inline-block" />
          {item.priority_second_approver_name}
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

  {#snippet actions(item: PurchaseOrdersAugmentedResponse)}
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
</DsList>
