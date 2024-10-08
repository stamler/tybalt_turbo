<script lang="ts">
  import Icon from "@iconify/svelte";
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import type { PurchaseOrdersResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import { shortDate } from "$lib/utilities";
  import { invalidate } from "$app/navigation";

  let rejectModal: RejectModal;

  let { data }: { data: PageData } = $props();
  let items = $state(data.items);

  async function refresh() {
    await invalidate("app:purchaseOrders");
    items = data.items;
  }

  async function del(id: string): Promise<void> {
    // return immediately if items is not an array
    if (!Array.isArray(items)) return;

    try {
      await pb.collection("purchase_orders").delete(id);

      // remove the deleted item from the list
      items = items.filter((item) => item.id !== id);
    } catch (error: any) {
      alert(error.data.message);
    }
  }

  async function approve(id: string): Promise<void> {
    try {
      const response = await pb.send(`/api/purchase_orders/${id}/approve`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      // replace the item in the list with the updated item
      items = items?.map((item) => {
        if (item.id === id) {
          return response as PurchaseOrdersResponse;
        }
        return item;
      });
    } catch (error: any) {
      globalStore.addError(error?.response?.message);
    }
  }

  function openRejectModal(poId: string) {
    rejectModal?.openModal(poId);
  }
</script>

{#snippet anchor(item: PurchaseOrdersResponse)}
  <span class="flex flex-col items-center">
    {#if item.status === "Unapproved"}
      {item.date}
    {:else}
      {item.po_number}
    {/if}
    <DsActionButton
      action={() => navigator.clipboard.writeText(JSON.stringify(item))}
      icon="mdi:clipboard-outline"
      title="Copy"
      color="blue"
    />
  </span>
{/snippet}

{#snippet headline({ total, payment_type, vendor_name }: PurchaseOrdersResponse)}
  <span class="flex items-center gap-2">
    ${total}
    {payment_type}
    <span class="flex items-center gap-0">
      <Icon icon="mdi:store" width="24px" class="inline-block" />
      {vendor_name}
    </span>
  </span>
{/snippet}

{#snippet byline({
  description,
  rejected,
  expand,
  rejection_reason,
  status,
}: PurchaseOrdersResponse)}
  <span class="flex items-center gap-2">
    {description}
    {#if rejected !== ""}
      <DsLabel color="red" title={`${shortDate(rejected)}: ${rejection_reason}`}>
        <Icon icon="mdi:cancel" width="24px" class="inline-block" />
        {expand?.rejector.expand?.profiles_via_uid.given_name}
        {expand?.rejector.expand?.profiles_via_uid.surname}
      </DsLabel>
    {:else if status === "Active"}
      <DsLabel color="green">
        <Icon icon="mdi:check" width="24px" class="inline-block" />
      </DsLabel>
    {:else if status === "Cancelled"}
      <DsLabel color="orange">
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
      {item.expand.job.number} - {item.expand.job.client}
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
    {:else if item.type === "Cumulative"}
      <DsLabel color="teal">
        <!-- <Icon icon="mdi:chart-bell-curve-cumulative" width="24px" class="inline-block" /> -->
        <Icon icon="mdi:sigma" width="24px" class="inline-block" />
      </DsLabel>
    {/if}
    {#if item.approved !== ""}
      <Icon icon="material-symbols:order-approve-outline" width="24px" class="inline-block" />
      {item.expand.approver.expand?.profiles_via_uid.given_name}
      {item.expand.approver.expand?.profiles_via_uid.surname}
      ({shortDate(item.approved)})
    {/if}
    {#if item.second_approver !== "" && item.second_approval !== ""}
      <span class="flex items-center gap-1">
        /
        <Icon icon="material-symbols:order-approve-outline" width="24px" class="inline-block" />
        {item.expand.second_approver.expand?.profiles_via_uid.given_name}
        {item.expand.second_approver.expand?.profiles_via_uid.surname}
        as {item.second_approver_claim.toUpperCase()} ({shortDate(item.second_approval)})
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

{#snippet actions({ id, status }: PurchaseOrdersResponse)}
  {#if status === "Active"}
    <DsActionButton
      action={`/expenses/add/${id}`}
      icon="mdi:add-bold"
      title="Create Expense"
      color="green"
    />
  {/if}
  {#if status === "Unapproved"}
    <DsActionButton action={`/pos/${id}/edit`} icon="mdi:edit-outline" title="Edit" color="blue" />
    <DsActionButton action={() => approve(id)} icon="mdi:approve" title="Approve" color="green" />
    <DsActionButton
      action={() => openRejectModal(id)}
      icon="mdi:cancel"
      title="Reject"
      color="orange"
    />
    <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
  {/if}
{/snippet}

<RejectModal on:refresh={refresh} collectionName="purchase_orders" bind:this={rejectModal} />
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
