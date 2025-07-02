<script lang="ts">
  import type { PageData } from "./$types";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import Icon from "@iconify/svelte";
  import { shortDate } from "$lib/utilities";

  export let data: PageData;
</script>

<div class="mx-auto space-y-6 p-4">
  <!-- Header with edit button -->
  <div class="flex items-center gap-2">
    <h1 class="text-2xl font-bold">
      {data.vendor.name}
      {#if data.vendor.alias}
        <span class="opacity-60">({data.vendor.alias})</span>
      {/if}
    </h1>
    <DsActionButton
      action={`/vendors/${data.vendor.id}/edit`}
      icon="mdi:pencil"
      title="Edit Vendor"
      color="blue"
    />
  </div>

  <!-- Section with tabs -->
  <section class="space-y-2">
    <!-- Tabs -->
    <div class="flex gap-4 border-b pb-1">
      <a
        href={`?tab=purchase_orders&poPage=${data.poPage}&expPage=${data.expPage}`}
        class={`pb-1 ${data.tab === "purchase_orders" ? "border-b-2 font-semibold" : ""}`}
      >
        Purchase Orders ({data.counts.purchase_orders})
      </a>
      <a
        href={`?tab=expenses&poPage=${data.poPage}&expPage=${data.expPage}`}
        class={`pb-1 ${data.tab === "expenses" ? "border-b-2 font-semibold" : ""}`}
      >
        Expenses ({data.counts.expenses})
      </a>
    </div>

    <div class="flex items-center justify-between">
      <h2 class="font-semibold">Page {data.page} / {data.totalPages}</h2>
      <div class="flex gap-2">
        {#if data.page > 1}
          <a
            href={`?tab=${data.tab}&poPage=${data.tab === "purchase_orders" ? data.page - 1 : data.poPage}&expPage=${data.tab === "expenses" ? data.page - 1 : data.expPage}`}
            class="rounded bg-neutral-200 px-2 py-1 hover:bg-neutral-300">&larr; Prev</a
          >
        {/if}
        {#if data.page < data.totalPages}
          <a
            href={`?tab=${data.tab}&poPage=${data.tab === "purchase_orders" ? data.page + 1 : data.poPage}&expPage=${data.tab === "expenses" ? data.page + 1 : data.expPage}`}
            class="rounded bg-neutral-200 px-2 py-1 hover:bg-neutral-300">Next &rarr;</a
          >
        {/if}
      </div>
    </div>

    <!-- Lists -->
    {#if data.tab === "purchase_orders"}
      <DsList items={data.purchaseOrders} search={false}>
        {#snippet anchor(po)}
          <a href={`/pos/${po.id}/details`} class="text-blue-600 hover:underline">
            {po.po_number}
          </a>
        {/snippet}
        {#snippet headline({ total, payment_type }: any)}
          ${total} {payment_type}
        {/snippet}
        {#snippet byline(po)}
          <span class="flex items-center gap-1">
            {shortDate(po.date)}
            {#if po.status === "Active"}
              <DsLabel style="inverted" color="green">{po.status}</DsLabel>
            {/if}
            {#if po.type === "Cumulative"}
              <DsLabel color="teal"><Icon icon="mdi:sigma" width="20" /></DsLabel>
            {/if}
          </span>
        {/snippet}
      </DsList>
    {:else}
      <DsList items={data.expenses} search={false}>
        {#snippet anchor(ex)}
          <a href={`/expenses/${ex.id}/details`} class="text-blue-600 hover:underline">
            {shortDate(ex.date)}
          </a>
        {/snippet}
        {#snippet headline(ex)}
          {ex.description || "Expense"}
        {/snippet}
        {#snippet byline({ total })}
          <span>${total}</span>
        {/snippet}
      </DsList>
    {/if}
  </section>
</div>
