<script lang="ts">
  import type { PageData } from "./$types";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsList from "$lib/components/DSList.svelte";
  import DSTabBar from "$lib/components/DSTabBar.svelte";
  import { shortDate } from "$lib/utilities";

  export let data: PageData;
  const d = data as any; // widen for newly added fields (owner tab)
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
            href={`?tab=${data.tab}&projectsPage=${data.tab === "projects" ? data.page - 1 : data.projectsPage}&proposalsPage=${data.tab === "proposals" ? data.page - 1 : data.proposalsPage}&ownerPage=${data.tab === "owner" ? data.page - 1 : d.ownerPage}`}
            class="rounded bg-neutral-200 px-2 py-1 hover:bg-neutral-300"
          >
            &larr; Prev
          </a>
        {/if}
        {#if data.page < data.totalPages}
          <a
            href={`?tab=${data.tab}&projectsPage=${data.tab === "projects" ? data.page + 1 : data.projectsPage}&proposalsPage=${data.tab === "proposals" ? data.page + 1 : data.proposalsPage}&ownerPage=${data.tab === "owner" ? data.page + 1 : d.ownerPage}`}
            class="rounded bg-neutral-200 px-2 py-1 hover:bg-neutral-300"
          >
            Next &rarr;
          </a>
        {/if}
      </div>
    </div>

    <DsList items={data.jobs} search={false}>
      {#snippet anchor(job)}
        <a href={`/jobs/${job.id}/details`} class="text-blue-600 hover:underline">
          {job.number}
        </a>
      {/snippet}
      {#snippet headline(job)}
        {job.description}
      {/snippet}
      {#snippet byline(job)}
        <span class="opacity-60">{shortDate(job.created)}</span>
      {/snippet}
    </DsList>
  </section>

  <!-- Contacts section -->
  <section class="rounded bg-neutral-100 p-2">
    <h2 class="mb-2 font-semibold">Contacts</h2>
    {#if data.contacts.length > 0}
      <div class="flex flex-wrap gap-1">
        {#each data.contacts as c}
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
