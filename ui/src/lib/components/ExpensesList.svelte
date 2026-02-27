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
  import { expensesEditingEnabled } from "$lib/stores/appConfig";
  import { onMount, onDestroy, untrack } from "svelte";
  import { proxySubscriptionWithLoader } from "$lib/utilities";
  import { type UnsubscribeFunc } from "pocketbase";

  const collectionId = "expenses";
  const viewerId = pb.authStore.record?.id ?? "";

  let {
    inListHeader,
    data,
    endpoint,
  }: {
    inListHeader?: string;
    data: { items: any; createdItemIsVisible?: any; totalPages?: number; limit?: number };
    endpoint: string;
  } = $props();
  let items = $state(untrack(() => data.items));
  let createdItemIsVisible = $state(untrack(() => data.createdItemIsVisible));
  let page = $state(1);
  let listLoading = $state(false);
  let hasMore = $state(untrack(() => (data.totalPages ?? 0) > 1));
  const serverLimit = untrack(() => data.limit ?? 20);

  // Subscribe to the base collection but update items using the details API
  let unsubscribeFunc: UnsubscribeFunc;
  onMount(async () => {
    if (items === undefined) {
      return;
    }
    unsubscribeFunc = await proxySubscriptionWithLoader<
      ExpensesResponse,
      ExpensesAugmentedResponse
    >(
      items,
      "expenses",
      async (id: string) => await pb.send(`/api/expenses/details/${id}`, { method: "GET" }),
      (newItems) => {
        items = newItems;
      },
      createdItemIsVisible,
    );
  });
  onDestroy(async () => {
    unsubscribeFunc();
  });

  async function loadMore() {
    if (listLoading || !hasMore) return;
    listLoading = true;
    try {
      const nextPage = page + 1;
      const res: { data: ExpensesAugmentedResponse[]; total_pages?: number; limit?: number } =
        await pb.send(`${endpoint}?page=${nextPage}&limit=20`, { method: "GET" });
      const newItems = res?.data ?? [];
      // Append and advance
      items = [...items, ...newItems];
      page = nextPage;
      const limit = res?.limit ?? serverLimit;
      const nextTotalPages = res?.total_pages;
      hasMore =
        nextTotalPages !== undefined ? nextPage < nextTotalPages : newItems.length === limit;
    } catch (e) {
      // noop, leave hasMore as-is
    } finally {
      listLoading = false;
    }
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

  async function approve(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/approve`, {
        method: "POST",
      });
    } catch (error: any) {
      globalStore.addError(error?.response.error);
    }
  }
</script>

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
  {#snippet actions({ id, uid, approver, submitted, approved, rejected, committed }: ExpensesAugmentedResponse)}
    {#if $expensesEditingEnabled}
      {@const isOwner = uid === viewerId}
      {@const isApprover = approver === viewerId}
      {@const hasTaprAccess = $globalStore.claims.includes("tapr")}
      {#if isOwner && !submitted}
        <DsActionButton
          action={`/expenses/${id}/edit`}
          icon="mdi:edit-outline"
          title="Edit"
          color="blue"
        />
      {/if}
      {#if isOwner && ((submitted && approved === "") || rejected !== "") && committed === ""}
        <DsActionButton action={() => recall(id)} icon="mdi:rewind" title="Recall" color="orange" />
      {/if}
      {#if isOwner && isApprover && hasTaprAccess && submitted && approved === "" && rejected === "" && committed === ""}
        <DsActionButton action={() => approve(id)} icon="mdi:approve" title="Approve" color="green" />
      {/if}
      {#if isOwner && !submitted}
        <DsActionButton action={() => submit(id)} icon="mdi:send" title="Submit" color="blue" />
      {/if}
      <!--
        Most review actions remain disabled in list views to encourage users to
        open expense details before reject/commit. Keep commented for easy
        rollback if policy changes.
      -->
      <!--
      {#if submitted && approved === ""}
        <DsActionButton
          action={() => approve(id)}
          icon="mdi:approve"
          title="Approve"
          color="green"
        />
      {/if}
      {#if approved !== "" && rejected === "" && committed === ""}
        <DsActionButton
          action={() => openRejectModal(id)}
          icon="mdi:cancel"
          title="Reject"
          color="orange"
        />
      {/if}
      {#if committed === "" && approved !== ""}
        <DsActionButton
          action={() => commit(id)}
          icon="mdi:check-all"
          title="Commit"
          color="green"
        />
      {/if}
      -->
      <!-- Delete is only available for unsubmitted, uncommitted owner records -->
      {#if isOwner && committed !== ""}
        <DsLabel color="green">Committed</DsLabel>
      {:else if isOwner && !submitted}
        <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
      {/if}
    {:else if uid === viewerId && committed !== ""}
      <DsLabel color="green">Committed</DsLabel>
    {/if}
  {/snippet}
</DsList>
{#if hasMore}
  <div class="mt-4 text-center">
    <button
      class="mb-4 rounded-sm bg-blue-600 px-4 py-2 text-white"
      onclick={loadMore}
      disabled={listLoading}
    >
      {listLoading ? "Loading…" : "Load More"}
    </button>
  </div>
{/if}
