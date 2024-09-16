<script lang="ts">
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import type { PurchaseOrdersResponse } from "$lib/pocketbase-types";
  import { shortDate } from "$lib/utilities";

  let { data }: { data: PageData } = $props();
  let items = $state(data.items);

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
</script>

{#snippet anchor(item: PurchaseOrdersResponse)}{item.date}{/snippet}

{#snippet headline({ total, payment_type, vendor_name }: PurchaseOrdersResponse)}
  <span>${total} {payment_type} / {vendor_name}</span>
{/snippet}

{#snippet byline({ description }: PurchaseOrdersResponse)}
  <span>{description}</span>
{/snippet}

{#snippet line1({ expand }: PurchaseOrdersResponse)}
  <span>
    {expand?.uid.expand?.profiles_via_uid.given_name}
    {expand?.uid.expand?.profiles_via_uid.surname} / {expand?.division.name}
    division
  </span>
{/snippet}

{#snippet line2(item: PurchaseOrdersResponse)}
  {#if item.approved !== ""}
    <span>
      Approved by
      {item.expand.approver.expand?.profiles_via_uid.given_name}
      {item.expand.approver.expand?.profiles_via_uid.surname}
      on {shortDate(item.approved)}
    </span>
  {/if}
  {#if item.second_approver !== "" && item.second_approval !== ""}
    <span>
      / Approved by
      {item.expand.second_approver.expand?.profiles_via_uid.given_name}
      {item.expand.second_approver.expand?.profiles_via_uid.surname}
      as {item.second_approver_claim.toUpperCase()} on {shortDate(item.second_approval)}
    </span>
  {/if}
{/snippet}
{#snippet line3(item: PurchaseOrdersResponse)}
  <!-- if the item is recurring, show the frequency -->
  {#if item.type === "Recurring"}
    <DsLabel color="cyan">
      Recurring {item.frequency} until {shortDate(item.end_date, true)}
    </DsLabel>
  {:else if item.type === "Cumulative"}
    <DsLabel color="teal">Cumulative</DsLabel>
  {/if}
  <button onclick={() => navigator.clipboard.writeText(JSON.stringify(item))}>
    Copy JSON to clipboard
  </button>
  {#if item.attachment}
    <a
      href={`${PUBLIC_POCKETBASE_URL}/api/files/${item.collectionId}/${item.id}/${item.attachment}`}
      target="_blank"
    >
      <DsFileLink filename={item.attachment as string} />
    </a>
  {/if}
{/snippet}

{#snippet actions({ id }: PurchaseOrdersResponse)}
  <a href="/pos/{id}/edit">edit</a>
  <button type="button" onclick={() => del(id)}>delete</button>
{/snippet}

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
