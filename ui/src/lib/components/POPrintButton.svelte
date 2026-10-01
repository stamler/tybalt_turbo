<script lang="ts">
  import Icon from "@iconify/svelte";
  import { resolve } from "$app/paths";
  import { fetchVisiblePO, type VisiblePurchaseOrderResponse } from "$lib/poVisibility";
  import {
    authorizedAmountText,
    defaultPOPrintOptions,
    poPrintAcknowledgementKey,
    poPrintSearchParams,
    poPrintStorageKey,
    poPrintWarning,
    printMoney,
    validatePOPrintOptions,
    type POPrintOptions,
  } from "$lib/poPrint";

  let { poId }: { poId: string } = $props();
  let dialog: HTMLDialogElement;
  let po = $state<VisiblePurchaseOrderResponse | null>(null);
  let options = $state<POPrintOptions>({ amount: "", plusTax: false, plusShipping: false });
  let loading = $state(false);
  let loadError = $state("");
  let printError = $state("");
  let acknowledgedKey = $state("");
  let request = 0;
  const validationError = $derived(po ? validatePOPrintOptions(po, options) : null);
  const warning = $derived(po ? poPrintWarning(po, options) : null);
  const acknowledgementKey = $derived(po ? poPrintAcknowledgementKey(po, options) : "");
  const acknowledged = $derived(acknowledgedKey === acknowledgementKey);

  // Editing any option requires a new acknowledgement, even if it is later restored.
  function resetAcknowledgement() {
    acknowledgedKey = "";
  }

  async function openOptions() {
    const currentRequest = ++request;
    po = null;
    options = { amount: "", plusTax: false, plusShipping: false };
    acknowledgedKey = "";
    loadError = "";
    printError = "";
    loading = true;
    if (!dialog.open) dialog.showModal();
    try {
      const currentPO = await fetchVisiblePO(poId);
      if (currentRequest !== request) return;
      po = currentPO;
      options = defaultPOPrintOptions(currentPO);
    } catch {
      if (currentRequest === request) loadError = "Could not load the purchase order. Try again.";
    } finally {
      if (currentRequest === request) loading = false;
    }
  }

  function closeOptions() {
    request++;
    dialog.close();
  }

  function proceed(event: SubmitEvent) {
    event.preventDefault();
    if (!po || loading || validationError || (warning && !acknowledged)) return;
    // Set tab-local options before navigation, then remove access to the opener.
    const printTab = window.open("about:blank", "_blank");
    if (!printTab) {
      printError = "Allow pop-ups for this site, then select Proceed again.";
      return;
    }
    try {
      const token = crypto.randomUUID();
      printTab.sessionStorage.setItem(
        poPrintStorageKey(poId, token),
        poPrintSearchParams(po, options).toString(),
      );
      printTab.opener = null;
      printTab.location.replace(`${resolve(`/pos/${poId}/print`)}?print=${token}`);
      closeOptions();
    } catch {
      printTab.close();
      printError = "Could not open the print page. Allow tab storage for this site and try again.";
    }
  }
</script>

<button
  type="button"
  class="inline-flex items-center gap-2 rounded-xs bg-blue-200 px-3 py-1 text-neutral-700 hover:bg-blue-300 hover:text-blue-500 active:bg-blue-800"
  onclick={openOptions}
>
  <Icon icon="mdi:printer-outline" width="20px" />
  Print
</button>

<dialog
  bind:this={dialog}
  aria-labelledby="po-print-title"
  class="m-auto max-h-[90dvh] w-[calc(100%-2rem)] max-w-lg overflow-y-auto rounded-lg border border-neutral-300 bg-white p-6 text-neutral-900 shadow-xl backdrop:bg-black/50"
  oncancel={closeOptions}
>
  <form onsubmit={proceed} novalidate>
    <h2 id="po-print-title" class="mb-4 text-xl font-bold">Print options</h2>
    {#if loading}
      <p role="status">Loading purchase order…</p>
    {:else if loadError}
      <p role="alert" class="text-red-700">{loadError}</p>
      <button type="button" class="mt-2 underline" onclick={openOptions}>Retry</button>
    {:else if po}
      <div class="space-y-4">
        <div>
          <div class="font-semibold">
            {po.type === "Cumulative" ? "Remaining available balance" : "Approved amount"}
          </div>
          <div>
            {printMoney(po.print_max_amount, po.currency_code)}
            {#if po.type === "Recurring"}
              / {po.frequency || "period"}{/if}
          </div>
          {#if po.type === "Cumulative"}
            <p class="text-sm text-neutral-600">After committed expenses.</p>
          {/if}
        </div>
        <div>
          <label for="po-print-amount" class="mb-1 block font-semibold"
            >Amount to print ({po.currency_code})</label
          >
          <input
            id="po-print-amount"
            type="text"
            inputmode="decimal"
            class="w-full rounded-sm border border-neutral-400 px-3 py-2"
            bind:value={options.amount}
            oninput={resetAcknowledgement}
            aria-invalid={Boolean(validationError)}
            aria-describedby={validationError ? "po-print-error" : undefined}
          />
          {#if validationError}
            <p id="po-print-error" role="alert" class="mt-1 text-sm text-red-700">
              {validationError}
            </p>
          {/if}
        </div>
        <div class="flex flex-wrap gap-4">
          <label class="flex items-center gap-2"
            ><input
              type="checkbox"
              bind:checked={options.plusTax}
              onchange={resetAcknowledgement}
            /> Plus tax</label
          >
          <label class="flex items-center gap-2"
            ><input
              type="checkbox"
              bind:checked={options.plusShipping}
              onchange={resetAcknowledgement}
            /> Plus shipping</label
          >
        </div>
        <p class="text-sm text-neutral-600">
          Unchecked charges are included in the printed amount, if applicable.
        </p>
        {#if warning}
          <div class="rounded-sm border border-amber-400 bg-amber-50 p-3">
            <p role="alert">{warning}</p>
            <label class="mt-3 flex items-start gap-2">
              <input
                type="checkbox"
                class="mt-1"
                checked={acknowledged}
                onchange={(event) => {
                  acknowledgedKey = event.currentTarget.checked ? acknowledgementKey : "";
                }}
              />
              I understand that extra costs above this allowance require a new approved PO.
            </label>
          </div>
        {/if}
        <div class="rounded-sm border border-neutral-300 p-3" aria-live="polite">
          <div class="font-semibold">Authorized Amount</div>
          <div data-testid="po-print-preview">{authorizedAmountText(po, options)}</div>
        </div>
      </div>
    {/if}
    {#if printError}<p role="alert" class="mt-3 text-red-700">{printError}</p>{/if}
    <div class="mt-6 flex justify-end gap-2">
      <button
        type="button"
        class="rounded-sm bg-neutral-200 px-4 py-2 hover:bg-neutral-300"
        onclick={closeOptions}>Cancel</button
      >
      <button
        type="submit"
        class="rounded-sm bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:opacity-50"
        disabled={!po || loading || Boolean(validationError) || Boolean(warning && !acknowledged)}
        >Proceed</button
      >
    </div>
  </form>
</dialog>
