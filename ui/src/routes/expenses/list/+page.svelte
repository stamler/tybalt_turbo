<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { PageData } from "./$types";
  import type { ExpensesResponse } from "$lib/pocketbase-types";
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
    <span>
      ${item.total} / {item.vendor_name}
    </span>
  {/snippet}
  {#snippet line2(item: ExpensesResponse)}{JSON.stringify(item)}{/snippet}
  {#snippet actions({ id }: ExpensesResponse)}
    <DsActionButton
      action={`/time/entries/${id}/edit`}
      icon="mdi:edit-outline"
      title="Edit"
      color="blue"
    />
    <DsActionButton action={() => del(id)} icon="mdi:delete" title="Delete" color="red" />
  {/snippet}
</DsList>
