<script lang="ts">
  import Icon from "@iconify/svelte";
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import type { ExpensesAugmentedResponse, ExpensesResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";
  import { shortDate } from "$lib/utilities";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import { onMount, onDestroy } from "svelte";
  import { augmentedProxySubscription } from "$lib/utilities";
  import { type UnsubscribeFunc } from "pocketbase";
  let rejectModal: RejectModal;

  const collectionId = "expenses";

  let {
    inListHeader,
    data,
  }: { inListHeader?: string; data: { items: any; createdItemIsVisible?: any } } = $props();
  let items = $state(data.items);
  let createdItemIsVisible = $state(data.createdItemIsVisible);

  // Subscribe to the base collection but update the items from the augmented
  // view
  let unsubscribeFunc: UnsubscribeFunc;
  onMount(async () => {
    if (items === undefined) {
      return;
    }
    unsubscribeFunc = await augmentedProxySubscription<ExpensesResponse, ExpensesAugmentedResponse>(
      items,
      "expenses",
      "expenses_augmented",
      (newItems) => {
        items = newItems;
      },
      createdItemIsVisible,
    );
  });
  onDestroy(async () => {
    unsubscribeFunc();
  });

  async function del(id: string): Promise<void> {
    // return immediately if items is not an array
    if (!Array.isArray(items)) return;

    try {
      await pb.collection("expenses").delete(id);

      // remove the item from the list
      items = items.filter((item) => item.id !== id);
    } catch (error: any) {
      alert(error.data.message);
    }
  }
  async function approve(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/approve`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });
    } catch (error: any) {
      globalStore.addError(error?.response.error);
    }
  }

  async function commit(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/commit`, {
        method: "POST",
      });
    } catch (error: any) {
      globalStore.addError(error?.response.error);
    }
  }

  async function submit(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/submit`, {
        method: "POST",
      });
    } catch (error: any) {
      globalStore.addError(error?.response.error);
    }
  }

  async function recall(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/recall`, {
        method: "POST",
      });
    } catch (error: any) {
      globalStore.addError(error?.response.error);
    }
  }

  function openRejectModal(recordId: string) {
    rejectModal?.openModal(recordId);
  }
</script>

<RejectModal collectionName="expenses" bind:this={rejectModal} />
<DsList
  items={items as ExpensesAugmentedResponse[]}
  search={true}
  {inListHeader}
  groupField="pay_period_ending"
>
  {#snippet groupHeader(field: string)}
    Pay Period Ending {field}
  {/snippet}
  {#snippet anchor(item: ExpensesAugmentedResponse)}
    <a href={`/expenses/${item.id}/details`} class="text-blue-600 hover:underline">
      {item.date}
    </a>
  {/snippet}
  {#snippet headline(item: ExpensesAugmentedResponse)}
    <span>{item.description}</span>
  {/snippet}
  {#snippet byline({
    rejector_name,
    rejected,
    rejection_reason,
    distance,
    total,
    payment_type,
    vendor,
    vendor_name,
    vendor_alias,
    cc_last_4_digits,
  }: ExpensesAugmentedResponse)}
    <span class="flex items-center gap-2 text-sm">
      {#if rejected !== ""}
        <DsLabel color="red" title={`${shortDate(rejected)}: ${rejection_reason}`}>
          <Icon icon="mdi:cancel" width="20px" class="inline-block" />
          {rejector_name}
        </DsLabel>
      {/if}

      {#if payment_type === "Mileage"}
        {distance} km / ${total}
      {:else}
        ${total}
      {/if}
      {#if vendor}
        <span class="flex items-center gap-0">
          <Icon icon="mdi:store" width="20px" class="inline-block" />
          {vendor_name}
          {#if vendor_alias}
            <span class="text-xs text-gray-500">({vendor_alias})</span>
          {/if}
        </span>
      {/if}
      {#if payment_type === "CorporateCreditCard"}
        <DsLabel color="cyan">
          <Icon icon="mdi:credit-card-outline" width="20px" class="inline-block" />
          **** {cc_last_4_digits}
        </DsLabel>
      {/if}
    </span>
  {/snippet}
  {#snippet line1({ uid_name, division_code, division_name }: ExpensesAugmentedResponse)}
    <span>
      {uid_name} / {division_code}
      {division_name}
    </span>
  {/snippet}
  {#snippet line2({
    job,
    job_number,
    job_description,
    client_name,
    category,
    category_name,
  }: ExpensesAugmentedResponse)}
    {#if job !== ""}
      <span class="flex items-center gap-1">
        {job_number} - {client_name}:
        {job_description}
        {#if category !== undefined}
          <DsLabel color="teal">{category_name}</DsLabel>
        {/if}
      </span>
    {/if}
  {/snippet}
  {#snippet line3({ approved, approver_name, attachment, id }: ExpensesAugmentedResponse)}
    <span class="flex items-center gap-1 text-sm">
      {#if approved !== ""}
        <Icon icon="material-symbols:order-approve-outline" width="20px" class="inline-block" />
        {approver_name}
        ({shortDate(approved)})
      {/if}
      {#if attachment}
        <a
          href={`${PUBLIC_POCKETBASE_URL}/api/files/${collectionId}/${id}/${attachment}`}
          target="_blank"
        >
          <DsFileLink filename={attachment as string} />
        </a>
      {/if}
    </span>
  {/snippet}
  {#snippet actions({ id, submitted, approved, rejected, committed }: ExpensesAugmentedResponse)}
    {#if !submitted}
      <DsActionButton
        action={`/expenses/${id}/edit`}
        icon="mdi:edit-outline"
        title="Edit"
        color="blue"
      />
    {/if}
    {#if (submitted && approved === "") || rejected !== ""}
      <DsActionButton action={() => recall(id)} icon="mdi:rewind" title="Recall" color="orange" />
    {/if}
    {#if !submitted}
      <DsActionButton action={() => submit(id)} icon="mdi:send" title="Submit" color="blue" />
    {/if}
    {#if submitted && approved === ""}
      <DsActionButton action={() => approve(id)} icon="mdi:approve" title="Approve" color="green" />
    {/if}
    <!-- Approved records can be rejected if they haven't been committed or rejected already -->
    {#if approved !== "" && rejected === "" && committed === ""}
      <DsActionButton
        action={() => openRejectModal(id)}
        icon="mdi:cancel"
        title="Reject"
        color="orange"
      />
    {/if}
    <!-- Commit button is disabled if the record has already been committed or is not approved -->
    {#if committed === "" && approved !== ""}
      <DsActionButton action={() => commit(id)} icon="mdi:check-all" title="Commit" color="green" />
    {/if}
    <!-- Delete button is disabled if the record has already been committed -->
    {#if committed !== ""}
      <DsLabel color="green">Committed</DsLabel>
    {:else}
      <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
    {/if}
  {/snippet}
</DsList>
