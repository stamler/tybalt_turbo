<script lang="ts">
  import { onMount } from "svelte";
  import type { PageData } from "./$types";
  import { shortDate, trimmedOrEmpty } from "$lib/utilities";

  let { data }: { data: PageData } = $props();

  const po = $derived(data.po);
  const currencyFormatter = new Intl.NumberFormat("en-CA", {
    style: "currency",
    currency: "CAD",
  });

  function formatCurrency(value: number): string {
    return currencyFormatter.format(value);
  }

  function displayValue(value: string | null | undefined, fallback = "Not specified"): string {
    return trimmedOrEmpty(value) || fallback;
  }

  const vendorAlias = $derived(trimmedOrEmpty(po.vendor_alias));
  const vendorLine = $derived(
    vendorAlias ? `${displayValue(po.vendor_name)} (${vendorAlias})` : displayValue(po.vendor_name),
  );

  const recurringDetails = $derived(
    po.type === "Recurring"
      ? [trimmedOrEmpty(po.frequency), po.end_date ? `Until ${shortDate(po.end_date, true)}` : ""]
          .filter(Boolean)
          .join(" / ")
      : "",
  );
  const recurringPeriodLabel = $derived(trimmedOrEmpty(po.frequency) || "period");
  const authorizedAmountText = $derived(
    po.type === "Recurring"
      ? `${formatCurrency(po.total)} / ${recurringPeriodLabel}`
      : formatCurrency(po.total),
  );

  onMount(() => {
    document.title = po.po_number ? `Purchase Order ${po.po_number}` : "Purchase Order";

    const timer = setTimeout(() => window.print(), 300);
    return () => clearTimeout(timer);
  });
</script>

<svelte:head>
  <meta name="robots" content="noindex" />
</svelte:head>

<div class="screen-only mx-auto max-w-5xl px-4 pt-4 pb-0">
  <div
    class="rounded-sm border border-neutral-300 bg-neutral-100 px-4 py-2 text-sm text-neutral-700"
  >
    The print dialog should open automatically. If it does not, use your browser's print command to
    save this purchase order as a PDF.
  </div>
</div>

<article class="print-page mx-auto my-4 max-w-5xl bg-white p-8 text-black">
  <!-- Header -->
  <div class="border-b-2 border-black pb-4" style="overflow: hidden;">
    <div class="border border-black px-4 py-3 text-sm" style="float: right; min-width: 225px;">
      <div class="font-semibold uppercase">Issue Date</div>
      <div class="mt-1">{shortDate(po.date, true)}</div>
      <div class="mt-3 font-semibold uppercase">Status</div>
      <div class="mt-1">{po.status}</div>
    </div>
    <div
      style="display: grid; grid-template-columns: auto 1fr; align-items: center; gap: 0.75rem; margin-right: 1.5rem;"
    >
      <img src="/logo.svg" alt="TBT Engineering Limited" class="h-12 w-12" />
      <div>
        <div class="text-lg font-bold tracking-wide">TBT Engineering Limited</div>
        <div class="text-sm font-semibold tracking-[0.2em] uppercase">Purchase Order</div>
      </div>
    </div>
    <h1 class="mt-2 text-3xl font-bold">{displayValue(po.po_number, "Pending Number")}</h1>
    <p class="mt-1 text-sm text-neutral-700">
      This document confirms authorization for goods or services up to the stated amount.
    </p>
  </div>

  <!-- Vendor + Summary -->
  <table
    style="width: calc(100% + 2rem); margin: 1rem -1rem 0; border-collapse: separate; border-spacing: 1rem 0;"
  >
    <tbody>
      <tr>
        <td style="width: 50%; vertical-align: top; border: 1px solid black; padding: 1rem;">
          <div class="text-xs font-semibold tracking-[0.2em] text-neutral-600 uppercase">
            Vendor
          </div>
          <div class="mt-2 text-xl font-semibold">{vendorLine}</div>
        </td>
        <td style="vertical-align: top; border: 1px solid black; padding: 1rem;">
          <div class="text-xs font-semibold tracking-[0.2em] text-neutral-600 uppercase">
            Purchase Order Summary
          </div>
          <table class="mt-2 text-sm" style="width: 100%; border-collapse: collapse;">
            <tbody>
              <tr>
                <td style="padding: 0.25rem 0.5rem 0.25rem 0; vertical-align: top; width: 50%;">
                  <div class="font-semibold">PO Type</div>
                  <div>{po.type}</div>
                </td>
                <td style="padding: 0.25rem 0; vertical-align: top;">
                  <div class="font-semibold">Authorized Amount</div>
                  <div>{authorizedAmountText}</div>
                </td>
              </tr>
              <tr>
                <td style="padding: 0.25rem 0.5rem 0.25rem 0; vertical-align: top;">
                  <div class="font-semibold">Owner</div>
                  <div>{displayValue(po.uid_name)}</div>
                </td>
                <td style="padding: 0.25rem 0; vertical-align: top;">
                  <div class="font-semibold">Payment Type</div>
                  <div>{displayValue(po.payment_type)}</div>
                </td>
              </tr>
              <tr>
                <td colspan="2" style="padding: 0.25rem 0; vertical-align: top;">
                  <div class="font-semibold">Division</div>
                  <div>
                    {#if trimmedOrEmpty(po.division_code) && trimmedOrEmpty(po.division_name)}
                      {po.division_code} - {po.division_name}
                    {:else}
                      {displayValue(po.division_name || po.division)}
                    {/if}
                  </div>
                </td>
              </tr>
              {#if recurringDetails}
                <tr>
                  <td colspan="2" style="padding: 0.25rem 0; vertical-align: top;">
                    <div class="font-semibold">Recurring Schedule</div>
                    <div>{recurringDetails}</div>
                  </td>
                </tr>
              {/if}
              {#if po.type === "Recurring"}
                <tr>
                  <td colspan="2" style="padding: 0.25rem 0; vertical-align: top;">
                    <div class="font-semibold">Maximum Authorized Total</div>
                    <div>{formatCurrency(po.approval_total)}</div>
                  </td>
                </tr>
              {/if}
            </tbody>
          </table>
        </td>
      </tr>
    </tbody>
  </table>

  <!-- Description + Job/Client -->
  <table
    style="width: calc(100% + 2rem); margin: 1rem -1rem 0; border-collapse: separate; border-spacing: 1rem 0;"
  >
    <tbody>
      <tr>
        <td style="width: 60%; vertical-align: top; border: 1px solid black; padding: 1rem;">
          <div class="text-xs font-semibold tracking-[0.2em] text-neutral-600 uppercase">
            Description
          </div>
          <div class="mt-2 text-sm leading-6 whitespace-pre-wrap">
            {displayValue(po.description)}
          </div>
        </td>
        <td style="vertical-align: top; border: 1px solid black; padding: 1rem;">
          <div class="text-xs font-semibold tracking-[0.2em] text-neutral-600 uppercase">
            Job / Client Context
          </div>
          <div class="mt-2 space-y-2 text-sm">
            {#if trimmedOrEmpty(po.job_number)}
              <div>
                <div class="font-semibold">Job</div>
                <div>
                  {po.job_number}
                  {#if trimmedOrEmpty(po.job_description)}
                    - {po.job_description}
                  {/if}
                </div>
              </div>
            {/if}

            {#if trimmedOrEmpty(po.client_name)}
              <div>
                <div class="font-semibold">Client</div>
                <div>{po.client_name}</div>
              </div>
            {/if}

            {#if trimmedOrEmpty(po.category_name)}
              <div>
                <div class="font-semibold">Category</div>
                <div>{po.category_name}</div>
              </div>
            {/if}

            {#if !trimmedOrEmpty(po.job_number) && !trimmedOrEmpty(po.client_name) && !trimmedOrEmpty(po.category_name)}
              <div>None provided.</div>
            {/if}
          </div>
        </td>
      </tr>
    </tbody>
  </table>

  <!-- Invoice Instructions -->
  <div class="p-4" style="border: 1px solid black; margin-top: 1rem;">
    <div class="text-xs font-semibold tracking-[0.2em] text-neutral-600 uppercase">
      Invoice Instructions
    </div>
    <p class="mt-2 text-sm leading-6">
      Please include purchase order {displayValue(po.po_number, "number")} on all invoices and supporting
      documents. Submit invoices through your billing contact and reach out to your project representative
      if billing details need to be confirmed.
    </p>
  </div>
</article>

<style>
  @page {
    margin: 1.25cm;
    size: auto;
  }

  .print-page {
    box-shadow: none;
  }

  @media print {
    .screen-only {
      display: none !important;
    }

    .print-page {
      margin: 0 !important;
      max-width: 100% !important;
      padding: 0.5cm !important;
      background: white !important;
    }
  }
</style>
