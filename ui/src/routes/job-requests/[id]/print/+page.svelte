<script lang="ts">
  import { onMount, tick } from "svelte";
  import {
    contactEmail,
    fullName,
    address,
    billingAddress,
    wholeCAD,
    printMissing,
  } from "$lib/jobRequests";
  let { data } = $props();
  const r = $derived(data.detail.request),
    d = $derived(data.detail.related);
  const missing = $derived(printMissing(data.detail));
  const shown = (value: any) =>
    value === undefined || value === null || value === "" ? "Not provided" : value;
  const date = (value: string) => (value ? value.slice(0, 10) : "Not provided");
  const rate = (value: number) =>
    Number.isFinite(value) ? `$${value.toFixed(2)}` : "Not provided";
  onMount(() => {
    let cancelled = false;
    (async () => {
      await tick();
      await document.fonts.ready;
      await Promise.all(Array.from(document.images).map((img) => img.decode().catch(() => {})));
      if (!cancelled) window.print();
    })();
    return () => {
      cancelled = true;
    };
  });
</script>

<svelte:head
  ><title>Project Authorization - {r.id}</title><meta
    name="robots"
    content="noindex"
  /></svelte:head
>
<div class="screen-only mx-auto max-w-4xl space-y-2 p-4">
  <a class="text-blue-700 underline" href={`/job-requests/${r.id}`}>Back to request</a>
  <p class="rounded border bg-neutral-100 p-3">
    The print dialog should open automatically. Use your browser's print command to print or Save as
    PDF.
  </p>
</div>
<article class="print-page">
  <table class="document">
    <thead
      ><tr
        ><td
          ><div class="reference">
            Request reference: {r.id} · Version {r.version} · Generated: {data.generated}
          </div></td
        ></tr
      ></thead
    ><tbody
      ><tr
        ><td>
          <header>
            <img src="/logo.svg" alt="TBT Engineering Limited" />
            <div>
              <strong>TBT Engineering Limited</strong>
              <h1>Project Authorization</h1>
            </div>
          </header>
          <p>Project number: To be assigned</p>
          {#if missing.length}<section class="draft">
              <h2>DRAFT - NOT FOR SIGNATURE</h2>
              <p>Complete or correct these items before signature:</p>
              <ul>
                {#each missing as value}<li>{value}</li>{/each}
              </ul>
            </section>{/if}
          <section>
            <h2>Client</h2>
            <p><strong>{shown(d.client?.name)}</strong></p>
            {#if address(d.client)}<p>
                {address(d.client)}
              </p>{/if}{#if d.job_owner?.id && d.job_owner.id !== d.client?.id}<p>
                Project owner: {d.job_owner.name}
              </p>{/if}
          </section>
          <section>
            <h2>Project / scope of work</h2>
            <p class="preserve">{shown(r.description)}</p>
            <p><strong>Location (Plus Code):</strong> {shown(r.location)}</p>
          </section>
          {#if d.proposal || d.parent || r.client_po || r.client_reference_number}<section>
              <h2>References</h2>
              {#if d.proposal}<p>
                  Proposal: {d.proposal.number} — {d.proposal.description}
                </p>{/if}{#if d.parent}<p>
                  Parent project: {d.parent.number} — {d.parent.description}
                </p>{/if}{#if r.client_po}<p>
                  Client PO: {r.client_po}
                </p>{/if}{#if r.client_reference_number}<p>
                  Client reference number: {r.client_reference_number}
                </p>{/if}
            </section>{/if}
          <section>
            <h2>Project contacts</h2>
            <div class="columns">
              <div>
                <h3>Client project contact</h3>
                <p>{shown(fullName(d.contact))}</p>
                <p>{shown(contactEmail(d.contact))}</p>
                {#if d.contact?.phone}<p>{d.contact.phone}</p>{/if}
              </div>
              <div>
                <h3>TBT project manager</h3>
                <p>{shown(fullName(d.manager))}</p>
                {#if d.manager?.email}<p>{d.manager.email}</p>{/if}{#if d.manager?.phone}<p>
                    {d.manager.phone}
                  </p>{/if}
                <p>Branch: {shown(d.branch?.name)}</p>
              </div>
            </div>
          </section>
          <section>
            <h2>Fees</h2>
            <p class="value">
              <strong>Project value (CAD, before tax): {wholeCAD(r.project_value)}</strong>
            </p>
            <p>
              Fee basis: {r.time_and_materials_set
                ? r.time_and_materials
                  ? "Time and materials"
                  : "Fixed fee"
                : "Not provided"}
            </p>
            <p>Applicable taxes are additional.</p>
            <p>
              Rate sheet: {shown(d.rate_sheet?.name)} · Revision: {shown(d.rate_sheet?.revision)} · Effective:
              {date(d.rate_sheet?.effective_date)}
            </p>
          </section>
          <section>
            <h2>Invoicing</h2>
            <div class="columns">
              <div>
                <p><strong>Bill to:</strong> {shown(d.client?.name)}</p>
                <p>Billing contact: {shown(fullName(d.billing_contact))}</p>
                <p>{shown(contactEmail(d.billing_contact))}</p>
                {#if d.billing_contact?.phone}<p>
                    Phone: {d.billing_contact.phone}
                  </p>{/if}{#if d.invoicing_information?.fax}<p>
                    Billing fax: {d.invoicing_information.fax}
                  </p>{/if}
              </div>
              <div>
                {#if billingAddress(d.billing_contact, d.client)}<p>
                    {billingAddress(d.billing_contact, d.client)}
                  </p>{/if}{#if d.invoicing_information?.invoicing_instructions}<p class="preserve">
                    {d.invoicing_information.invoicing_instructions}
                  </p>{/if}
              </div>
            </div>
          </section>
          {#if !missing.length}<section>
              <h2>Agreement and signatures</h2>
              <p>
                The client authorizes TBT Engineering Limited to perform the work described above on
                the stated fee basis and agrees to the stated project value and invoicing details. {#if r.time_and_materials}For
                  time and materials work, the attached rate sheet applies.
                {/if}Applicable taxes are additional.
              </p>
              <div class="columns signatures">
                {#each [`For the client: ${d.client.name}`, "For TBT Engineering Limited"] as title}<div
                    class="signature"
                  >
                    <h3>{title}</h3>
                    {#each ["Printed name", "Title", "Signature", "Date"] as label}<div
                        class="signature-line"
                      >
                        <span>{label}</span>
                      </div>{/each}
                  </div>{/each}
              </div>
            </section>{/if}
          {#if r.time_and_materials}<section class="appendix">
              <h2>Client billing rate sheet — CAD</h2>
              <h3>{shown(d.rate_sheet?.name)} — Revision {shown(d.rate_sheet?.revision)}</h3>
              <p>Effective: {date(d.rate_sheet?.effective_date)}</p>
              <table class="rates">
                <thead
                  ><tr><th>Role</th><th>Regular hourly rate</th><th>Overtime hourly rate</th></tr
                  ></thead
                ><tbody
                  >{#each d.rate_sheet_entries || [] as entry}<tr
                      ><td
                        >{entry.role_name ||
                          entry.expand?.role?.name ||
                          entry.role?.name ||
                          "Not provided"}</td
                      ><td>{rate(entry.rate)}</td><td>{rate(entry.overtime_rate)}</td></tr
                    >{:else}<tr
                      ><td colspan="3"
                        >Client billing rates are not available. DRAFT - NOT FOR SIGNATURE.</td
                      ></tr
                    >{/each}</tbody
                >
              </table>
            </section>{/if}
        </td></tr
      ></tbody
    >
  </table>
</article>

<style>
  .print-page {
    max-width: 8.5in;
    margin: 1rem auto;
    padding: 0.5in;
    color: black;
    background: white;
    font:
      11pt/1.25 Arial,
      sans-serif;
  }
  .document {
    width: 100%;
    border-collapse: collapse;
  }
  .document > thead {
    display: table-header-group;
  }
  .reference {
    border-bottom: 1px solid black;
    padding-bottom: 0.08in;
    margin-bottom: 0.14in;
    font-size: 9pt;
  }
  header {
    display: flex;
    align-items: center;
    gap: 0.15in;
  }
  header img {
    width: 0.48in;
    height: 0.48in;
  }
  h1 {
    font-size: 20pt;
    margin: 0;
  }
  h2 {
    font-size: 12pt;
    font-weight: bold;
    border-bottom: 1px solid black;
    margin: 0 0 0.06in;
  }
  h3 {
    font-size: 11pt;
    font-weight: bold;
    margin: 0;
  }
  section {
    margin-top: 0.1in;
  }
  p {
    margin: 0.035in 0;
    overflow-wrap: anywhere;
  }
  .preserve {
    white-space: pre-wrap;
  }
  .columns {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.3in;
  }
  .value {
    font-size: 13pt;
  }
  .draft {
    border: 2px solid black;
    padding: 0.1in;
  }
  .draft h2 {
    border: 0;
  }
  ul {
    margin: 0.05in 0;
    padding-left: 0.2in;
  }
  .signature {
    break-inside: avoid;
    page-break-inside: avoid;
  }
  .signatures {
    margin-top: 0.1in;
  }
  .signature-line {
    border-bottom: 1px solid black;
    height: 0.32in;
    position: relative;
    margin-bottom: 0.12in;
  }
  .signature-line span {
    position: absolute;
    bottom: -0.15in;
    font-size: 9pt;
  }
  .appendix {
    break-before: page;
    page-break-before: always;
  }
  .rates {
    width: 100%;
    border-collapse: collapse;
    margin-top: 0.15in;
  }
  .rates th,
  .rates td {
    padding: 0.08in;
    border-bottom: 1px solid black;
    text-align: right;
  }
  .rates th:first-child,
  .rates td:first-child {
    text-align: left;
  }
  .rates tr {
    break-inside: avoid;
  }
  @page {
    size: letter portrait;
    margin: 0.5in;
  }
  @media print {
    .screen-only {
      display: none !important;
    }
    header {
      display: flex !important;
    }
    header img {
      filter: grayscale(1);
    }
    .columns {
      display: grid !important;
      grid-template-columns: 1fr 1fr;
    }
    .signature-line {
      height: 0.32in !important;
      position: relative !important;
    }
    .print-page {
      margin: 0;
      padding: 0;
      max-width: none;
    }
    :global(body) {
      background: white !important;
    }
    * {
      color: black !important;
      background: white !important;
    }
  }
</style>
