<script module>
  let idCounter = $state(0);
</script>

<script lang="ts">
  import type { CurrenciesResponse } from "$lib/pocketbase-types";
  import {
    currencyIconHref,
    formatCurrencyEquivalent,
    normalizeCurrencyCode,
  } from "$lib/utilities";

  const thisId = idCounter;
  idCounter += 1;

  let {
    amount = $bindable(),
    currency = $bindable(),
    items = [] as CurrenciesResponse[],
    errors = {} as Record<string, { message: string }>,
    amountFieldName = "total",
    currencyFieldName = "currency",
    uiName = "Total",
    amountStep = 0.01,
    amountMin = 0,
    disabledAmount = false,
    disabledCurrency = false,
    helperText = "",
    homeEquivalent,
    rate,
    rateDate,
    displayCode = "",
    displaySymbol = "",
  }: {
    amount: number | string;
    currency: string;
    items?: CurrenciesResponse[];
    errors?: Record<string, { message: string }>;
    amountFieldName?: string;
    currencyFieldName?: string;
    uiName?: string;
    amountStep?: number;
    amountMin?: number;
    disabledAmount?: boolean;
    disabledCurrency?: boolean;
    helperText?: string;
    homeEquivalent?: number | null;
    rate?: number | null;
    rateDate?: string | null;
    displayCode?: string;
    displaySymbol?: string;
  } = $props();

  const selectedCurrency = $derived.by(
    () =>
      items.find((item) => item.id === currency) ??
      items.find((item) => item.code === normalizeCurrencyCode(currency)),
  );
  const selectCurrencyValue = $derived.by(() =>
    items.length === 0 ? currency : (selectedCurrency?.id ?? currency),
  );
  const previewCode = $derived.by(() =>
    selectedCurrency?.code ?? (displayCode.trim() !== "" ? displayCode : normalizeCurrencyCode(currency)),
  );
  const previewSymbol = $derived.by(() =>
    selectedCurrency?.symbol ?? (displaySymbol.trim() !== "" ? displaySymbol : previewCode),
  );
  const previewEquivalent = $derived.by(() =>
    homeEquivalent !== undefined && homeEquivalent !== null
      ? formatCurrencyEquivalent(homeEquivalent, rate, rateDate)
      : "",
  );

  const hasError = $derived(
    errors[amountFieldName] !== undefined || errors[currencyFieldName] !== undefined,
  );
</script>

<div class="flex w-full flex-col gap-2 {hasError ? 'bg-red-200' : ''}">
  <label for={`currency-input-${thisId}`}>{uiName}</label>
  <div
    class="flex items-center overflow-hidden rounded border border-neutral-300 bg-white focus-within:ring-2 focus-within:ring-blue-500"
  >
    <!-- Currency selector (left) -->
    {#if selectedCurrency?.icon}
      <img
        src={currencyIconHref(selectedCurrency.id, selectedCurrency.icon)}
        alt={`${previewCode} icon`}
        class="ml-1 h-6 w-6 rounded-full border border-neutral-200 object-cover"
      />
    {/if}
    <select
      name={currencyFieldName}
      value={selectCurrencyValue}
      onchange={(event) => {
        currency = (event.currentTarget as HTMLSelectElement).value;
      }}
      class="border-none bg-transparent py-1 pl-2 pr-1 text-sm font-semibold focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
      disabled={disabledCurrency}
    >
      {#if items.length === 0}
        <option value={currency}>{previewCode}</option>
      {:else}
        {#each items as item (item.id)}
          <option value={item.id}>{item.code}</option>
        {/each}
      {/if}
    </select>

    <!-- Divider -->
    <span class="h-5 w-px bg-neutral-300"></span>

    <!-- Amount input (center) -->
    <input
      id={`currency-input-${thisId}`}
      name={amountFieldName}
      type="number"
      bind:value={amount}
      min={amountMin}
      step={amountStep}
      class="min-w-0 flex-1 border-none bg-transparent px-2 py-1 focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
      disabled={disabledAmount}
    />

    <!-- Symbol badge (right) -->
    <span class="border-l border-neutral-300 bg-neutral-100 px-2 py-1 text-sm text-neutral-600">
      {previewSymbol}
    </span>
  </div>

  {#if errors[amountFieldName]}
    <span class="text-sm text-red-600">{errors[amountFieldName].message}</span>
  {/if}
  {#if errors[currencyFieldName]}
    <span class="text-sm text-red-600">{errors[currencyFieldName].message}</span>
  {/if}

  {#if helperText || previewEquivalent}
    <div class="text-sm text-neutral-600">
      {#if helperText}
        <div>{helperText}</div>
      {/if}
      {#if previewEquivalent}
        <div>CAD equivalent: {previewEquivalent}</div>
      {/if}
    </div>
  {/if}
</div>
