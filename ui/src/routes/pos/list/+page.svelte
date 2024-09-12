<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import type { PurchaseOrdersResponse } from "$lib/pocketbase-types";

  let { data }: { data: PageData } = $props();

  async function del(id: string): Promise<void> {
    // return immediately if data.items is not an array
    if (!Array.isArray(data.items)) return;

    try {
      await pb.collection("purchase_orders").delete(id);

      // remove the item from the list
      data.items = data.items.filter((item) => item.id !== id);
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

{#snippet line2(item: PurchaseOrdersResponse)}{JSON.stringify(item)}{/snippet}

{#snippet actions({ id }: PurchaseOrdersResponse)}
  <a href="/pos/{id}/edit">edit</a>
  <button type="button" onclick={() => del(id)}>delete</button>
{/snippet}

<DsList
  items={data.items as PurchaseOrdersResponse[]}
  search={true}
  inListHeader="Purchase Orders"
  {anchor}
  {headline}
  {byline}
  {line1}
  {line2}
  {actions}
/>
