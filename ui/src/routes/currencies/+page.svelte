<script lang="ts">
  import DSList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsFileSelect from "$lib/components/DsFileSelect.svelte";
  import DSTextInput from "$lib/components/DSTextInput.svelte";
  import { pb } from "$lib/pocketbase";
  import type { CurrenciesResponse } from "$lib/pocketbase-types";
  import type { CurrencyInitStatus, CurrencyListRow } from "$lib/svelte-types";
  import { currencies } from "$lib/stores/currencies";
  import { currencyIconHref, formatCurrencyAmount } from "$lib/utilities";
  import { onMount } from "svelte";

  currencies.init();

  const defaultItem = {
    code: "",
    symbol: "",
    icon: "",
    rate: 1,
    rate_date: "",
    ui_sort: 0,
  };

  let rows = $state<CurrencyListRow[]>([]);
  let initStatus = $state<CurrencyInitStatus | null>(null);
  let loading = $state(true);
  let saving = $state(false);
  let errors = $state({} as Record<string, { message: string }>);
  let item = $state({ ...defaultItem } as Partial<CurrenciesResponse> & { icon: File | string });
  let editingId = $state("");

  async function refreshPage() {
    loading = true;
    try {
      const [nextRows, nextStatus] = await Promise.all([
        pb.send("/api/currencies", { method: "GET" }) as Promise<CurrencyListRow[]>,
        pb.send("/api/currencies/init_status", { method: "GET" }) as Promise<CurrencyInitStatus>,
      ]);
      rows = nextRows;
      initStatus = nextStatus;
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    await refreshPage();
  });

  function clearForm() {
    item = { ...defaultItem };
    editingId = "";
    errors = {};
  }

  async function editRow(row: CurrencyListRow) {
    const record = await pb.collection("currencies").getOne<CurrenciesResponse>(row.id);
    item = {
      ...record,
      icon: record.icon,
    };
    editingId = row.id;
    errors = {};
  }

  async function save() {
    saving = true;
    errors = {};
    try {
      const payload = {
        ...item,
        code: (item.code ?? "").trim().toUpperCase(),
        symbol: (item.symbol ?? "").trim(),
      };
      if (editingId) {
        await pb.collection("currencies").update(editingId, payload);
      } else {
        await pb.collection("currencies").create(payload);
      }
      await currencies.refresh();
      await refreshPage();
      clearForm();
    } catch (error: any) {
      errors = error?.data?.data ?? { global: { message: error?.response?.message ?? "Save failed" } };
    } finally {
      saving = false;
    }
  }

  async function deleteCurrency(id: string) {
    try {
      await pb.send(`/api/currencies/${id}`, { method: "DELETE" });
      await currencies.refresh();
      await refreshPage();
      if (editingId === id) {
        clearForm();
      }
    } catch (error: any) {
      errors = {
        global: { message: error?.response?.message ?? "Delete failed" },
      };
    }
  }

  async function initializeCadBackfill(id: string) {
    try {
      await pb.send(`/api/currencies/${id}/initialize_backfill`, { method: "POST" });
      await refreshPage();
    } catch (error: any) {
      errors = {
        global: { message: error?.response?.message ?? "Initialization failed" },
      };
    }
  }
</script>

<div class="space-y-4 p-4">
  <h1 class="text-2xl font-bold">Currencies</h1>

  {#if initStatus}
    <div class="rounded-sm border border-neutral-300 bg-neutral-50 p-3 text-sm">
      <div>Home currency ready: {initStatus.home_currency_ready ? "Yes" : "No"}</div>
      <div>Blank purchase orders: {initStatus.blank_purchase_orders}</div>
      <div>Blank expenses: {initStatus.blank_expenses}</div>
      <div class="text-neutral-600">
        Create `CAD`, then run the CAD initialization action once to backfill legacy rows.
      </div>
    </div>
  {/if}

  {#if loading}
    <div class="text-neutral-500">Loading currencies…</div>
  {:else}
    <DSList items={rows} inListHeader="Configured Currencies" search={true}>
      {#snippet anchor(row: CurrencyListRow)}
        {#if row.icon}
          <img
            src={currencyIconHref(row.id, row.icon)}
            alt={`${row.code} icon`}
            class="h-10 w-10 rounded-full border border-neutral-200 object-cover"
          />
        {:else}
          <span
            class="inline-flex min-w-14 items-center justify-center rounded-sm border border-neutral-300 bg-neutral-100 px-2 py-2 text-xs font-semibold"
          >
            {row.code}
          </span>
        {/if}
      {/snippet}
      {#snippet headline(row: CurrencyListRow)}
        {row.code} · {row.symbol}
      {/snippet}
      {#snippet line1(row: CurrencyListRow)}
        Rate: {row.rate > 0 ? row.rate.toFixed(4) : "Not set"}
        {#if row.rate_date}
          · {row.rate_date}
        {/if}
      {/snippet}
      {#snippet line2(row: CurrencyListRow)}
        UI sort: {row.ui_sort}
        {#if row.used_by_purchase_orders}
          · Used by purchase orders
        {/if}
        {#if row.used_by_expenses}
          · Used by expenses
        {/if}
      {/snippet}
      {#snippet actions(row: CurrencyListRow)}
        <DsActionButton action={() => editRow(row)} color="blue">Edit</DsActionButton>
        {#if row.code === "CAD"}
          <DsActionButton action={() => initializeCadBackfill(row.id)} color="green">
            Initialize CAD
          </DsActionButton>
        {/if}
        <DsActionButton action={() => deleteCurrency(row.id)} color="red">Delete</DsActionButton>
      {/snippet}
    </DSList>
  {/if}

  <form class="flex w-full flex-col gap-2 rounded-sm border border-neutral-300 bg-neutral-50 p-4">
    <h2 class="text-lg font-semibold">{editingId ? "Edit Currency" : "Add Currency"}</h2>
    <DSTextInput bind:value={item.code as string} {errors} fieldName="code" uiName="ISO Code" />
    <DSTextInput bind:value={item.symbol as string} {errors} fieldName="symbol" uiName="Symbol" />
    <DSTextInput
      bind:value={item.ui_sort as number}
      {errors}
      fieldName="ui_sort"
      uiName="UI Sort"
      type="number"
      step={1}
    />
    <DSTextInput
      bind:value={item.rate as number}
      {errors}
      fieldName="rate"
      uiName="Rate to CAD"
      type="number"
      step={0.0001}
      min={0}
    />
    <DSTextInput
      bind:value={item.rate_date as string}
      {errors}
      fieldName="rate_date"
      uiName="Rate Date"
      placeholder="YYYY-MM-DD"
    />
    <DsFileSelect bind:record={item} {errors} fieldName="icon" uiName="SVG Icon" />
    {#if item.rate}
      <div class="text-sm text-neutral-600">
        Example: {formatCurrencyAmount(10, item.code)} ≈ {formatCurrencyAmount(10 * Number(item.rate), "CAD")}
      </div>
    {/if}
    <div class="flex gap-2">
      <DsActionButton action={save} loading={saving} color="green">Save</DsActionButton>
      <DsActionButton action={clearForm}>Clear</DsActionButton>
    </div>
    {#if errors.global}
      <div class="text-red-600">{errors.global.message}</div>
    {/if}
  </form>
</div>
