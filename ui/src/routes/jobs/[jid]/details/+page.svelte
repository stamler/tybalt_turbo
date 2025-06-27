<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import type { PageData } from "./$types";

  export let data: PageData;
  // Use data.job directly (Svelte 5 without $:)

  function personName(person: any) {
    if (!person) return "";
    return `${person.given_name || person.name || ""} ${person.surname || ""}`.trim();
  }
</script>

<div class="mx-auto space-y-4 p-4">
  <h1 class="text-2xl font-bold">Job Details</h1>

  <div class="space-y-2 rounded bg-neutral-100 p-4">
    <div class="flex items-center gap-2">
      <span class="font-semibold">Job Number:</span>
      <span>{data.job.number}</span>
      {#if data.job.number?.startsWith("P")}
        <DsLabel color="yellow">proposal</DsLabel>
      {/if}
    </div>
    <div><span class="font-semibold">Description:</span> {data.job.description}</div>
    {#if data.job.status}
      <div><span class="font-semibold">Status:</span> {data.job.status}</div>
    {/if}

    {#if data.job.client}
      <div><span class="font-semibold">Client:</span> {data.job.client.name}</div>
    {/if}

    {#if data.job.contact && (data.job.contact.given_name || data.job.contact.surname)}
      <div><span class="font-semibold">Contact:</span> {personName(data.job.contact)}</div>
    {/if}

    {#if data.job.manager && (data.job.manager.given_name || data.job.manager.surname)}
      <div><span class="font-semibold">Manager:</span> {personName(data.job.manager)}</div>
    {/if}

    {#if data.job.alternate_manager && (data.job.alternate_manager.given_name || data.job.alternate_manager.surname)}
      <div>
        <span class="font-semibold">Alternate Manager:</span>
        {personName(data.job.alternate_manager)}
      </div>
    {/if}

    {#if data.job.job_owner && (data.job.job_owner.given_name || data.job.job_owner.surname)}
      <div><span class="font-semibold">Job Owner:</span> {personName(data.job.job_owner)}</div>
    {/if}

    {#if data.job.proposal_id}
      <div>
        <span class="font-semibold">Proposal:</span>
        <a href={`/jobs/${data.job.proposal_id}/details`} class="text-blue-600 hover:underline">
          {data.job.proposal_number || data.job.proposal_id}
        </a>
      </div>
    {/if}

    <div>
      <span class="font-semibold">FN Agreement:</span>
      {data.job.fn_agreement ? "Yes" : "No"}
    </div>

    {#if data.job.project_award_date}
      <div>
        <span class="font-semibold">Project Award Date:</span>
        {data.job.project_award_date}
      </div>
    {/if}

    {#if data.job.proposal_opening_date}
      <div>
        <span class="font-semibold">Proposal Opening Date:</span>
        {data.job.proposal_opening_date}
      </div>
    {/if}

    {#if data.job.proposal_submission_due_date}
      <div>
        <span class="font-semibold">Proposal Submission Due:</span>
        {data.job.proposal_submission_due_date}
      </div>
    {/if}

    {#if data.job.divisions && Array.isArray(data.job.divisions)}
      <div>
        <span class="font-semibold">Divisions:</span>
        {#each data.job.divisions as division, idx}
          {division.name} ({division.code}){idx < data.job.divisions.length - 1 ? ", " : ""}
        {/each}
      </div>
    {/if}

    {#if data.job.projects && data.job.projects.length > 0}
      <div>
        <span class="font-semibold">Projects:</span>
        {#each data.job.projects as p, i}
          <a href={`/jobs/${p.id}/details`} class="text-blue-600 hover:underline">{p.number}</a>{i <
          data.job.projects.length - 1
            ? ", "
            : ""}
        {/each}
      </div>
    {/if}
  </div>

  <div class="flex gap-2">
    <DsActionButton
      action={`/jobs/${data.job.id}/edit`}
      icon="mdi:pencil"
      title="Edit Job"
      color="blue"
    />
  </div>
</div>
