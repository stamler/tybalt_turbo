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
  import { shortDate } from "$lib/utilities";
  import { invalidate, goto } from "$app/navigation";

  let { data }: { data: PageData } = $props();
  let items = $state(data.items);

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
</script>

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
      {#if item.payment_type === "Mileage"}
        {item.distance} km
      {:else}
        ${item.total}
      {/if}
      {#if item.vendor_name !== ""}
        <span class="flex items-center gap-0">
          <Icon icon="mdi:store" width="24px" class="inline-block" />
          {item.vendor_name}
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
          {item.expand.job.number} - {item.expand.job.client}
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
  {#snippet actions({ id }: ExpensesResponse)}
    <DsActionButton
      action={`/expenses/${id}/edit`}
      icon="mdi:edit-outline"
      title="Edit"
      color="blue"
    />
    <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
  {/snippet}
</DsList>
