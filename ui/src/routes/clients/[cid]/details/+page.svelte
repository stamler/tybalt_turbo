<script lang="ts">
  import type { PageData } from "./$types";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsList from "$lib/components/DSList.svelte";
  import DSTabBar from "$lib/components/DSTabBar.svelte";
  import { shortDate, formatCurrency } from "$lib/utilities";
  import DSCollapsible from "$lib/components/DSCollapsible.svelte";
  import NoteForm from "$lib/components/NoteForm.svelte";
  import ClientNoteItem from "$lib/components/ClientNoteItem.svelte";
  import type { ClientNote } from "./+page";
  import type { JobApiResponse } from "$lib/stores/jobs";

  type ClientJob = JobApiResponse & { created?: string };

  const { data } = $props<{ data: PageData }>();
  const d = data as any; // widen for newly added fields (owner tab)
  let notes = $state<ClientNote[]>(data.notes ?? []);
  let showNoteForm = $state(false);

  function handleNoteCreated(note: ClientNote) {
    notes = [note, ...notes];
    showNoteForm = false;
  }

  const jobs = data.jobs as ClientJob[];

  const leadName =
    data.client.lead_surname && data.client.lead_given_name
      ? `${data.client.lead_surname}, ${data.client.lead_given_name}`
      : "Not assigned";
</script>

<div class="mx-auto space-y-6 p-4">
  <!-- Header with edit button -->
  <div class="flex items-center gap-2">
    <h1 class="text-2xl font-bold">{data.client.name}</h1>
    <DsActionButton
      action={`/clients/${data.client.id}/edit`}
      icon="mdi:pencil"
      title="Edit Client"
      color="blue"
    />
  </div>

  <!-- Summary section -->
  <section class="rounded bg-neutral-100 p-2">
    <h2 class="mb-2 font-semibold">Summary</h2>
    <div class="grid gap-2 sm:grid-cols-2">
      <div>
        <h3 class="text-sm font-semibold text-neutral-600">Outstanding Balance</h3>
        <p class="text-lg font-medium">
          {formatCurrency(data.client.outstanding_balance ?? 0)}
        </p>
        {#if data.client.outstanding_balance_date}
          <p class="text-xs text-neutral-500">
            As of {shortDate(data.client.outstanding_balance_date, true)}
          </p>
        {/if}
      </div>
      <div>
        <h3 class="text-sm font-semibold text-neutral-600">Business Development Lead</h3>
        <p class="text-lg font-medium">
          {leadName}
        </p>
      </div>
    </div>

    <DSCollapsible title="Notes" collapsed>
      {#snippet headerActions(isCollapsed)}
        {#if !isCollapsed}
          <DsActionButton
            icon={showNoteForm ? "mdi:minus" : "mdi:plus"}
            title={showNoteForm ? "Hide note form" : "Add note"}
            color="green"
            transparentBackground={true}
            action={() => (showNoteForm = !showNoteForm)}
          />
        {/if}
      {/snippet}

      {#snippet children()}
        <div class="mt-2 space-y-4">
          {#if showNoteForm}
            <NoteForm
              clientId={data.client.id}
              jobs={data.noteJobs}
              onCreated={handleNoteCreated}
            />
          {/if}

          {#if notes.length === 0}
            <p class="text-sm italic text-neutral-600">No notes yet.</p>
          {:else}
            <ul class="space-y-3">
              {#each notes as note (note.id)}
                <ClientNoteItem
                  created={note.created}
                  message={note.note}
                  author={note.author}
                  job={note.job}
                />
              {/each}
            </ul>
          {/if}
        </div>
      {/snippet}
    </DSCollapsible>
  </section>

  <!-- Jobs list section -->
  <section class="space-y-2">
    <!-- Tabs -->
    <DSTabBar
      tabs={[
        {
          label: `Projects (${data.counts.projects})`,
          href: `?tab=projects&projectsPage=${data.projectsPage}&proposalsPage=${data.proposalsPage}&ownerPage=${d.ownerPage}`,
          active: data.tab === "projects",
        },
        {
          label: `Proposals (${data.counts.proposals})`,
          href: `?tab=proposals&projectsPage=${data.projectsPage}&proposalsPage=${data.proposalsPage}&ownerPage=${d.ownerPage}`,
          active: data.tab === "proposals",
        },
        {
          label: `Jobs as Owner (${d.counts.owner})`,
          href: `?tab=owner&projectsPage=${data.projectsPage}&proposalsPage=${data.proposalsPage}&ownerPage=${d.ownerPage}`,
          active: data.tab === "owner",
        },
      ]}
    />

    <div class="flex items-center justify-between">
      <h2 class="font-semibold">Page {data.page} / {data.totalPages}</h2>
      <div class="flex gap-2">
        {#if data.page > 1}
          <a
            href={`?tab=${data.tab}&projectsPage=${
              data.tab === "projects" ? data.page - 1 : data.projectsPage
            }&proposalsPage=${
              data.tab === "proposals" ? data.page - 1 : data.proposalsPage
            }&ownerPage=${data.tab === "owner" ? data.page - 1 : d.ownerPage}`}
            class="rounded bg-neutral-200 px-2 py-1 hover:bg-neutral-300"
          >
            &larr; Prev
          </a>
        {/if}
        {#if data.page < data.totalPages}
          <a
            href={`?tab=${data.tab}&projectsPage=${
              data.tab === "projects" ? data.page + 1 : data.projectsPage
            }&proposalsPage=${
              data.tab === "proposals" ? data.page + 1 : data.proposalsPage
            }&ownerPage=${data.tab === "owner" ? data.page + 1 : d.ownerPage}`}
            class="rounded bg-neutral-200 px-2 py-1 hover:bg-neutral-300"
          >
            Next &rarr;
          </a>
        {/if}
      </div>
    </div>

    <DsList items={jobs} search={false}>
      {#snippet anchor(job)}
        <a href={`/jobs/${job.id}/details`} class="text-blue-600 hover:underline">
          {job.number}
        </a>
      {/snippet}
      {#snippet headline(job)}
        {job.description}
      {/snippet}
      {#snippet byline(job)}
        <span class="opacity-60">{job.created ? shortDate(job.created) : ""}</span>
      {/snippet}
    </DsList>
  </section>

  <!-- Contacts section -->
  <section class="rounded bg-neutral-100 p-2">
    <h2 class="mb-2 font-semibold">Contacts</h2>
    {#if data.client.contacts.length > 0}
      <div class="flex flex-wrap gap-1">
        {#each data.client.contacts as c}
          <a
            href={`mailto:${c.email}`}
            class="rounded-md px-1 hover:cursor-pointer hover:bg-neutral-300"
            title={c.email}
          >
            {c.given_name}
            {c.surname}
          </a>
        {/each}
      </div>
    {:else}
      <p class="italic">No contacts recorded.</p>
    {/if}
  </section>
</div>
