<script lang="ts">
  import { MediaQuery } from "svelte/reactivity";
  import DSCard from "$lib/components/DSCard.svelte";
  import DSCardGrid from "$lib/components/DSCardGrid.svelte";
  import DSCollapsible from "$lib/components/DSCollapsible.svelte";
  import DSField from "$lib/components/DSField.svelte";
  import DSFieldList from "$lib/components/DSFieldList.svelte";
  import DSLocationPicker from "$lib/components/DSLocationPicker.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import JobInvoicingCard from "$lib/components/jobs/JobInvoicingCard.svelte";
  import JobValueLine from "$lib/components/jobs/JobValueLine.svelte";
  import { clientLabel } from "$lib/clientWorkspace";
  import { JobValue, valueDate } from "$lib/jobValue.svelte";
  import { formatCurrency, shortDate } from "$lib/utilities";
  import { resolve } from "$app/paths";
  import type { JobDetails, JobDetailsPerson, JobRef } from "./types";

  // The job's facts as a grid of cards. On phones the cards sit in a
  // collapsed "Job details" block so the tabs stay near the top.
  let { job, jobValue }: { job: JobDetails; jobValue: JobValue | null } = $props();

  const wide = new MediaQuery("min-width: 768px");
  const isProposal = $derived(job.number?.startsWith("P") ?? false);

  function personName(person: JobDetailsPerson | null | undefined) {
    return `${person?.given_name ?? ""} ${person?.surname ?? ""}`.trim();
  }

  function date(value: string | undefined) {
    return value ? shortDate(value, true) : "";
  }

  const branch = $derived(
    job.branch_name
      ? job.branch_code
        ? `${job.branch_name} (${job.branch_code})`
        : job.branch_name
      : job.branch_code || job.branch_id || "",
  );
  const allocations = $derived(Array.isArray(job.allocations) ? job.allocations : []);
  const children = $derived(Array.isArray(job.children) ? job.children : []);
  const projects = $derived(Array.isArray(job.projects) ? job.projects : []);
  const categories = $derived(Array.isArray(job.categories) ? job.categories : []);
  const hasStructure = $derived(
    allocations.length > 0 || children.length > 0 || projects.length > 0 || categories.length > 0,
  );
  const hasDates = $derived(
    isProposal
      ? Boolean(job.proposal_opening_date || job.proposal_submission_due_date)
      : Boolean(job.project_award_date || job.project_completion_date || job.request_creator),
  );
</script>

{#snippet jobLinks(list: JobRef[])}
  {#each list as item, i (item.id)}<a
      class="link"
      href={resolve("/jobs/[jid]/details", { jid: item.id })}>{item.number}</a
    >{i < list.length - 1 ? ", " : ""}{/each}
{/snippet}

{#snippet cards()}
  <DSCardGrid>
    <DSCard title="Client & team">
      <DSFieldList>
        {#if job.client?.id}
          <DSField
            label="Client"
            value={clientLabel(job.client)}
            href={resolve("/clients/[cid]/details", { cid: job.client.id })}
          />
        {/if}
        <DSField label="Contact" value={personName(job.contact)} />
        {#if job.job_owner?.id}
          <DSField
            label="Job owner"
            value={clientLabel(job.job_owner)}
            href={resolve("/clients/[cid]/details", { cid: job.job_owner.id })}
          />
        {/if}
        <DSField label="Manager" value={personName(job.manager)} />
        <DSField label="Alternate manager" value={personName(job.alternate_manager)} />
        <DSField label="Branch" value={branch} />
      </DSFieldList>
    </DSCard>

    <DSCard title={isProposal ? "Value" : "Value & contract"}>
      <DSFieldList>
        {#if isProposal}
          <DSField label="Proposal value" value={formatCurrency(job.proposal_value ?? 0)} />
        {:else}
          <DSField label="Project value">
            <div class="flex flex-wrap items-center gap-2">
              {formatCurrency(job.project_value ?? 0)}
              {#if jobValue}<JobValueLine value={jobValue} />{/if}
            </div>
          </DSField>
          <DSField
            label="Outstanding balance"
            value={formatCurrency(job.outstanding_balance ?? 0)}
            hint={job.outstanding_balance_date
              ? `as of ${shortDate(job.outstanding_balance_date, true)}`
              : undefined}
          />
          {#if job.rate_sheet?.id}
            <DSField
              label="Rate sheet"
              value={`${job.rate_sheet.name} (rev. ${job.rate_sheet.revision})`}
              href={resolve("/rate-sheets/[id]/details", { id: job.rate_sheet.id })}
            />
          {/if}
          <DSField label="Client PO" value={job.client_po} />
          <DSField label="Client reference number" value={job.client_reference_number} />
        {/if}
        <DSField label="FN agreement" value={job.fn_agreement ? "Yes" : "No"} />
      </DSFieldList>
    </DSCard>

    {#if !isProposal}
      <JobInvoicingCard {job} />
    {/if}

    {#if hasStructure}
      <DSCard title="Structure">
        <DSFieldList>
          {#if projects.length > 0}
            <DSField label="Projects">{@render jobLinks(projects)}</DSField>
          {/if}
          {#if children.length > 0}
            <DSField label="Subjobs">{@render jobLinks(children)}</DSField>
          {/if}
          {#if allocations.length > 0}
            <DSField label="Divisions">
              <div class="flex flex-wrap gap-1">
                {#each allocations as a, i (a.division?.id ?? i)}
                  <DsLabel color="blue" size="sm" title={a.division?.code}
                    >{a.division?.name} · {a.hours} h</DsLabel
                  >
                {/each}
              </div>
            </DSField>
          {/if}
          {#if categories.length > 0}
            <DSField label="Categories">
              <div class="flex flex-wrap gap-1">
                {#each categories as category (category.id ?? category.name)}
                  <DsLabel color="teal" size="sm">{category.name}</DsLabel>
                {/each}
              </div>
            </DSField>
          {/if}
        </DSFieldList>
      </DSCard>
    {/if}

    {#if hasDates}
      <DSCard title="Dates">
        <DSFieldList>
          {#if isProposal}
            <DSField label="Opening date" value={date(job.proposal_opening_date)} />
            <DSField label="Submission due" value={date(job.proposal_submission_due_date)} />
          {:else}
            <DSField label="Awarded" value={date(job.project_award_date)} />
            <DSField label="Completion" value={date(job.project_completion_date)} />
            {#if job.request_creator}
              <DSField label="Project request">
                Requested by {job.request_creator_name || job.request_creator} on {valueDate(
                  job.request_submitted || job.request_created,
                )}; created by {job.created_by_name || job.created_by} on {valueDate(
                  job.creation_completed,
                )}
              </DSField>
            {/if}
          {/if}
        </DSFieldList>
      </DSCard>
    {/if}
  </DSCardGrid>

  {#if job.location}
    <DSCollapsible title="Location">
      <DSLocationPicker
        value={job.location}
        errors={{}}
        fieldName="location"
        disabled={true}
        readonly={true}
      />
    </DSCollapsible>
  {/if}
{/snippet}

{#if wide.current}
  {@render cards()}
{:else}
  <DSCollapsible title="Job details">{@render cards()}</DSCollapsible>
{/if}
