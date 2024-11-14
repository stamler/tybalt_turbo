<script lang="ts">
  import Icon from "@iconify/svelte";
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import type { PageData } from "./$types";
  import type { ExpensesResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";
  import { shortDate } from "$lib/utilities";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import { invalidate } from "$app/navigation";

  let rejectModal: RejectModal;

  let { data }: { data: PageData } = $props();
  let items = $state(data.items);

  async function refresh() {
    await invalidate("app:expenses");
    items = data.items;
  }

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
      await refresh();
    } catch (error: any) {
      globalStore.addError(error?.response.error);
    }
  }

  async function commit(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/commit`, {
        method: "POST",
      });
      await refresh();
    } catch (error: any) {
      globalStore.addError(error?.response.error);
    }
  }

  async function submit(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/submit`, {
        method: "POST",
      });
      await refresh();
    } catch (error: any) {
      globalStore.addError(error?.response.error);
    }
  }

  async function recall(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/recall`, {
        method: "POST",
      });
      await refresh();
    } catch (error: any) {
      globalStore.addError(error?.response.error);
    }
  }

  function openRejectModal(recordId: string) {
    rejectModal?.openModal(recordId);
  }
</script>

<RejectModal on:refresh={refresh} collectionName="expenses" bind:this={rejectModal} />
<DsList
  items={items as ExpensesResponse[]}
  search={true}
  inListHeader="Expenses"
  groupField="pay_period_ending"
>
  {#snippet groupHeader(field: string)}
    Pay Period Ending {field}
  {/snippet}
  {#snippet anchor(item: ExpensesResponse)}{item.date}{/snippet}
  {#snippet headline(item: ExpensesResponse)}
    <span>{item.description}</span>
  {/snippet}
  {#snippet byline(item: ExpensesResponse)}
    <span class="flex items-center gap-2">
      {#if item.rejected !== ""}
        <DsLabel color="red" title={`${shortDate(item.rejected)}: ${item.rejection_reason}`}>
          <Icon icon="mdi:cancel" width="24px" class="inline-block" />
          {item.expand?.rejector.expand?.profiles_via_uid.given_name}
          {item.expand?.rejector.expand?.profiles_via_uid.surname}
        </DsLabel>
      {/if}

      {#if item.payment_type === "Mileage"}
        {item.distance} km / ${item.total}
      {:else}
        ${item.total}
      {/if}
      {#if item.expand?.vendor}
        <span class="flex items-center gap-0">
          <Icon icon="mdi:store" width="24px" class="inline-block" />
          {item.expand?.vendor.name} ({item.expand?.vendor.alias})
        </span>
      {/if}
      {#if item.payment_type === "CorporateCreditCard"}
        <DsLabel color="cyan">
          <Icon icon="mdi:credit-card-outline" width="24px" class="inline-block" />
          **** {item.cc_last_4_digits}
        </DsLabel>
      {/if}
    </span>
  {/snippet}
  {#snippet line1(item: ExpensesResponse)}
    <span>
      {item.expand?.uid.expand?.profiles_via_uid.given_name}
      {item.expand?.uid.expand?.profiles_via_uid.surname}
      / {item.expand?.division.code}
      {item.expand?.division.name}
    </span>
  {/snippet}
  {#snippet line2(item: ExpensesResponse)}
    {#if item.job !== ""}
      {#if item.expand?.job}
        <span class="flex items-center gap-1">
          {item.expand.job.number} - {item.expand.job.expand.client.name}:
          {item.expand.job.description}
          {#if item.expand?.category !== undefined}
            <DsLabel color="teal">{item.expand?.category.name}</DsLabel>
          {/if}
        </span>
      {/if}
    {/if}
  {/snippet}
  {#snippet line3(item: ExpensesResponse)}
    <span class="flex items-center gap-1">
      {#if item.approved !== ""}
        <Icon icon="material-symbols:order-approve-outline" width="24px" class="inline-block" />
        {item.expand.approver.expand?.profiles_via_uid.given_name}
        {item.expand.approver.expand?.profiles_via_uid.surname}
        ({shortDate(item.approved)})
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
  {#snippet actions({ id, submitted, approved, rejected, committed }: ExpensesResponse)}
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
