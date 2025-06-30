<script lang="ts">
  import type { PageData } from "./$types";
  import DsActionButton from "$lib/components/DSActionButton.svelte";

  export let data: PageData;
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
    <div class="flex gap-4 border-b pb-1">
      <a
        href={`?tab=projects&projectsPage=${data.projectsPage}&proposalsPage=${data.proposalsPage}`}
        class={`pb-1 ${data.tab === "projects" ? "border-b-2 font-semibold" : ""}`}
      >
        Projects ({data.counts.projects})
      </a>
      <a
        href={`?tab=proposals&projectsPage=${data.projectsPage}&proposalsPage=${data.proposalsPage}`}
        class={`pb-1 ${data.tab === "proposals" ? "border-b-2 font-semibold" : ""}`}
      >
        Proposals ({data.counts.proposals})
      </a>
    </div>

    <div class="flex items-center justify-between">
      <h2 class="font-semibold capitalize">{data.tab} (page {data.page} / {data.totalPages})</h2>
      <div class="flex gap-2">
        {#if data.page > 1}
          <a
            href={`?tab=${data.tab}&projectsPage=${data.tab === "projects" ? data.page - 1 : data.projectsPage}&proposalsPage=${data.tab === "proposals" ? data.page - 1 : data.proposalsPage}`}
            class="rounded bg-neutral-200 px-2 py-1 hover:bg-neutral-300"
          >
            &larr; Prev
          </a>
        {/if}
        {#if data.page < data.totalPages}
          <a
            href={`?tab=${data.tab}&projectsPage=${data.tab === "projects" ? data.page + 1 : data.projectsPage}&proposalsPage=${data.tab === "proposals" ? data.page + 1 : data.proposalsPage}`}
            class="rounded bg-neutral-200 px-2 py-1 hover:bg-neutral-300"
          >
            Next &rarr;
          </a>
        {/if}
      </div>
    </div>

    <ul class="divide-y divide-neutral-200 rounded bg-neutral-100">
      {#if data.jobs.length > 0}
        {#each data.jobs as job}
          <li class="p-2">
            <a href={`/jobs/${job.id}/details`} class="text-blue-600 hover:underline">
              {job.number}
            </a>
            <span class="opacity-60"> — {job.description}</span>
          </li>
        {/each}
      {:else}
        <li class="p-2 italic">No jobs found.</li>
      {/if}
    </ul>
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
