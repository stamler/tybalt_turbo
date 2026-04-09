<script lang="ts">
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DSList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DSCurrencyInput from "$lib/components/DSCurrencyInput.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import type { ExpenseSettlementRow } from "$lib/svelte-types";
  import { pb } from "$lib/pocketbase";
  import {
    formatCurrencyAmount,
    formatCurrencyEquivalent,
    settlementToleranceBounds,
    shortDate,
    trimmedOrEmpty,
  } from "$lib/utilities";
  import { resolve } from "$app/paths";
  import { onMount } from "svelte";

  let activeTab = $state<"unsettled" | "settled">("unsettled");
  let unsettledRows = $state<ExpenseSettlementRow[]>([]);
  let settledRows = $state<ExpenseSettlementRow[]>([]);
  let draftValues = $state<Record<string, number>>({});
  let loading = $state(false);
  let errorMessage = $state("");

  const settlementTabs = [
    { id: "unsettled", label: "Unsettled" },
    { id: "settled", label: "Settled" },
  ] as const;

  function paymentTypeLabel(paymentType: string): string {
    switch (paymentType) {
      case "CorporateCreditCard":
        return "Corporate Credit Card";
      case "OnAccount":
        return "On Account";
      default:
        return paymentType;
    }
  }

  async function refreshRows() {
    loading = true;
    errorMessage = "";
    try {
      const [unsettled, settled] = await Promise.all([
        pb.send("/api/expenses/unsettled", { method: "GET" }) as Promise<ExpenseSettlementRow[]>,
        pb.send("/api/expenses/settled", { method: "GET" }) as Promise<ExpenseSettlementRow[]>,
      ]);
      unsettledRows = unsettled;
      settledRows = settled;
      draftValues = Object.fromEntries(
        unsettled.map((row) => [row.id, row.settled_total || 0]),
      );
    } catch (error: any) {
      errorMessage = error?.response?.message ?? "Failed to load settlement queue";
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    await refreshRows();
  });

  async function saveSettlement(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/settle`, {
        method: "POST",
        body: JSON.stringify({ settled_total: Number(draftValues[id] ?? 0) }),
        headers: { "Content-Type": "application/json" },
      });
      await refreshRows();
    } catch (error: any) {
      errorMessage = error?.response?.message ?? "Failed to save settlement";
    }
  }

  async function clearSettlement(id: string) {
    try {
      await pb.send(`/api/expenses/${id}/clear_settlement`, { method: "POST" });
      await refreshRows();
    } catch (error: any) {
      errorMessage = error?.response?.message ?? "Failed to clear settlement";
    }
  }

  const rows = $derived(activeTab === "unsettled" ? unsettledRows : settledRows);
</script>

{#if errorMessage}
  <div class="mx-2 mt-2 rounded-sm border border-red-300 bg-red-50 p-3 text-red-700">{errorMessage}</div>
{/if}

{#if loading}
  <div class="p-2 text-neutral-500">Loading settlement queue…</div>
{:else}
  <DSList items={rows} inListHeader={activeTab === "unsettled" ? "Unsettled" : "Settled"} search={true}>
    {#snippet searchBarExtra()}
      <div class="flex items-center gap-2 max-[639px]:w-full max-[639px]:flex-wrap">
        <DSToggle
          bind:value={activeTab}
          options={settlementTabs}
          ariaLabel="Expense settlement status filter"
        />
      </div>
    {/snippet}

    {#snippet anchor(row: ExpenseSettlementRow)}
      <a href={resolve(`/expenses/${row.id}/details`)} class="text-blue-600 hover:underline">
        {shortDate(row.date, true)}
      </a>
    {/snippet}
    {#snippet headline(row: ExpenseSettlementRow)}
      {row.creator_name} · {row.description}
    {/snippet}
    {#snippet byline(row: ExpenseSettlementRow)}
      <span class="flex items-center gap-2 text-sm">
        <DsLabel color="cyan">{paymentTypeLabel(row.payment_type)}</DsLabel>
        {#if row.payment_type === "CorporateCreditCard" && trimmedOrEmpty(row.cc_last_4_digits)}
          <DsLabel color="cyan">**** {row.cc_last_4_digits}</DsLabel>
        {/if}
      </span>
    {/snippet}
    {#snippet line1(row: ExpenseSettlementRow)}
      {formatCurrencyAmount(row.total, row.currency_code)}
      {#if row.po_number}
        · PO {row.po_number}
      {/if}
      {#if trimmedOrEmpty(row.vendor_name)}
        · {row.vendor_name}
      {/if}
    {/snippet}
    {#snippet line2(row: ExpenseSettlementRow)}
      {formatCurrencyEquivalent(row.indicative_cad_total, row.currency_rate, row.currency_rate_date)}
    {/snippet}
    {#snippet line3(row: ExpenseSettlementRow)}
      {#if activeTab === "unsettled"}
        {@const toleranceBounds = settlementToleranceBounds(row.indicative_cad_total)}
        <DSCurrencyInput
          bind:amount={draftValues[row.id]}
          currency=""
          items={[]}
          amountFieldName={`settled_total_${row.id}`}
          currencyFieldName={`currency_${row.id}`}
          uiName="Settled CAD Total"
          disabledCurrency={true}
          helperText={`Original amount: ${formatCurrencyAmount(row.total, row.currency_code)} · latest CAD equivalent ${formatCurrencyAmount(row.indicative_cad_total, "CAD")} · allowed range ${formatCurrencyAmount(toleranceBounds.min, "CAD")} to ${formatCurrencyAmount(toleranceBounds.max, "CAD")}`}
          displayCode="CAD"
          displaySymbol="CAD"
        />
      {:else}
        Settled {formatCurrencyAmount(row.settled_total, "CAD")}
        {#if row.settled}
          · {shortDate(row.settled, true)}
        {/if}
        {#if trimmedOrEmpty(row.settler_name)}
          · {row.settler_name}
        {/if}
      {/if}
    {/snippet}
    {#snippet actions(row: ExpenseSettlementRow)}
      {#if activeTab === "unsettled"}
        <DsActionButton action={() => saveSettlement(row.id)} color="green">Save</DsActionButton>
      {:else}
        <DsActionButton action={() => clearSettlement(row.id)} color="red">Clear</DsActionButton>
      {/if}
      <DsActionButton action={resolve(`/expenses/${row.id}/details`)}>View</DsActionButton>
    {/snippet}
  </DSList>
{/if}
