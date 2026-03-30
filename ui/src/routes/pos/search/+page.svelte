<script lang="ts">
  import Icon from "@iconify/svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DSSearchList from "$lib/components/DSSearchList.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import { pb } from "$lib/pocketbase";
  import type { POSearchApiResponse } from "$lib/stores/poSearch";
  import { poSearch } from "$lib/stores/poSearch";
  import { expensesEditingEnabled } from "$lib/stores/appConfig";
  import { globalStore } from "$lib/stores/global";
  import { pocketBaseFileHref, shortDate, trimmedOrEmpty } from "$lib/utilities";

  const collectionId = "purchase_orders";
  type SearchStatus = POSearchApiResponse["status"];

  const allStatusOptions: { id: SearchStatus; label: string }[] = [
    { id: "Active", label: "Active" },
    { id: "Closed", label: "Closed" },
    { id: "Cancelled", label: "Cancelled" },
  ];

  // Only show toggle options for statuses the user actually has visibility into.
  // The backend already filters by permission, so we derive from loaded data.
  const statusOptions = $derived(
    allStatusOptions.filter((opt) =>
      $poSearch.items.some((po: POSearchApiResponse) => po.status === opt.id),
    ),
  );

  let selectedStatus = $state<SearchStatus>("Active");

  poSearch.init();

  function poStatusColor(
    status: POSearchApiResponse["status"],
  ): "green" | "gray" | "yellow" | "red" {
    if (status === "Active") return "green";
    if (status === "Closed" || status === "Cancelled") return "gray";
    return "yellow";
  }

  function poMayBeCancelledByUser(po: POSearchApiResponse): boolean {
    return (
      po.status === "Active" &&
      $globalStore.claims.includes("payables_admin") &&
      po.committed_expenses_count === 0
    );
  }

  function poMayBeClosedByUser(po: POSearchApiResponse): boolean {
    return (
      po.status === "Active" &&
      po.type !== "One-Time" &&
      $globalStore.claims.includes("payables_admin") &&
      po.committed_expenses_count > 0
    );
  }

  async function cancel(id: string): Promise<void> {
    try {
      await pb.send(`/api/purchase_orders/${id}/cancel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
    } catch (error: any) {
      globalStore.addError(error?.response?.message ?? "Failed to cancel purchase order");
    }
  }

  async function closePurchaseOrder(id: string): Promise<void> {
    try {
      await pb.send(`/api/purchase_orders/${id}/close`, { method: "POST" });
    } catch (error: any) {
      globalStore.addError(error?.response?.message ?? "Failed to close purchase order");
    }
  }

  const statusFilter = $derived((po: POSearchApiResponse) => po.status === selectedStatus);
</script>

{#if $poSearch.index !== null}
  <DSSearchList
    index={$poSearch.index}
    filter={statusFilter}
    inListHeader={selectedStatus}
    fieldName="purchase_order"
    uiName="search purchase orders..."
    collectionName="purchase_orders_search"
  >
    {#snippet searchBarExtra()}
      <div class="flex items-center gap-2 max-[639px]:w-full max-[639px]:flex-wrap">
        <DSToggle
          bind:value={selectedStatus}
          options={statusOptions}
          ariaLabel="PO status filter"
        />
      </div>
    {/snippet}

    {#snippet anchor(item: POSearchApiResponse)}
      {#if item.status === "Active"}
        <DsLabel style="inverted" color="green">
          <a href={`/pos/${item.id}/details`} class="hover:underline">
            {item.po_number}
          </a>
        </DsLabel>
      {:else}
        <a href={`/pos/${item.id}/details`} class="text-blue-600 hover:underline">
          {item.po_number}
        </a>
      {/if}
    {/snippet}

    {#snippet headline({ vendor_name, vendor_alias }: POSearchApiResponse)}
      <span class="flex items-center gap-2">
        {vendor_name}
        {#if trimmedOrEmpty(vendor_alias)}
          ({trimmedOrEmpty(vendor_alias)})
        {/if}
      </span>
    {/snippet}

    {#snippet byline(item: POSearchApiResponse)}
      <span class="flex items-center gap-2">
        ${item.total}
        {#if item.legacy_manual_entry}
          <DsLabel color="cyan">Manually created</DsLabel>
        {/if}
        {#if item.status !== "Active"}
          <DsLabel color={poStatusColor(item.status)}>
            {item.status}
          </DsLabel>
        {/if}
      </span>
    {/snippet}

    {#snippet line1({ description }: POSearchApiResponse)}
      {description}
    {/snippet}

    {#snippet line2({
      job_number,
      client_name,
      job_description,
      category_name,
    }: POSearchApiResponse)}
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

    {#snippet line3(item: POSearchApiResponse)}
      <span class="flex items-center gap-1">
        {#if item.uid_name}
          <span class="text-sm text-slate-600">Owner: {item.uid_name}</span>
        {/if}
        {#if item.type === "Cumulative"}
          <DsLabel color="cyan">
            <Icon icon="mdi:sigma" width="24px" class="inline-block" />
          </DsLabel>
        {/if}
        {#if item.type === "Recurring"}
          <DsLabel color="cyan">
            <Icon icon="mdi:recurring-payment" width="24px" class="inline-block" />
            {item.frequency} until {shortDate(item.end_date, true)}
          </DsLabel>
        {/if}
        {#if item.attachment}
          <a href={pocketBaseFileHref(collectionId, item.id, item.attachment)} target="_blank">
            <DsFileLink filename={item.attachment} />
          </a>
        {/if}
      </span>
    {/snippet}

    {#snippet actions(item: POSearchApiResponse)}
      {#if $expensesEditingEnabled && item.status === "Active"}
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
    {/snippet}
  </DSSearchList>
{/if}
