<script lang="ts">
  import DSCard from "$lib/components/DSCard.svelte";
  import DSField from "$lib/components/DSField.svelte";
  import DSFieldList from "$lib/components/DSFieldList.svelte";
  import InvoicingInstructions from "$lib/components/clients/InvoicingInstructions.svelte";
  import { resolve } from "$app/paths";
  import type { JobDetails } from "./types";

  // Who receives a project's invoices and how, from the client invoicing
  // profile the project uses. A project without a profile says so, since
  // the gap needs fixing before it can be invoiced.
  let { job }: { job: JobDetails } = $props();

  const profileHref = $derived(
    job.invoicing_information && job.client?.id
      ? `${resolve("/clients/[cid]/details", { cid: job.client.id })}?show_invoicing=${job.invoicing_information}#invoicing`
      : "",
  );
</script>

<DSCard title="Invoicing">
  {#snippet actions()}
    {#if profileHref}
      <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- query and hash on a client page -->
      <a class="link text-sm" href={profileHref}>Open profile</a>
    {/if}
  {/snippet}
  {#if job.invoicing_information}
    <DSFieldList>
      <DSField label="Profile" value={job.invoicing_profile_name} />
      {#if job.invoice_contact_name || job.invoice_contact_email}
        <DSField label="Send invoices to">
          {job.invoice_contact_name}
          {#if job.invoice_contact_name && job.invoice_contact_email}·{/if}
          {#if job.invoice_contact_email}
            <a class="link" href={`mailto:${job.invoice_contact_email}`}
              >{job.invoice_contact_email}</a
            >
          {/if}
        </DSField>
      {/if}
    </DSFieldList>
    <InvoicingInstructions text={job.invoice_instructions || ""} hideWhenEmpty />
  {:else}
    <p class="text-sm text-neutral-600">No invoicing profile</p>
  {/if}
</DSCard>
