<script lang="ts">
  import DSCollapsible from "$lib/components/DSCollapsible.svelte";
  import DSList from "$lib/components/DSList.svelte";
  import { formatDateTime } from "$lib/utilities";
  import NoteForm from "$lib/components/NoteForm.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import type { ClientNote, NoteJobOption } from "$lib/types/notes";
  import { untrack } from "svelte";

  let {
    clientId,
    notes: initialNotes = [] as ClientNote[],
    jobOptions = [] as NoteJobOption[],
    preselectedJobId = "",
    heading = "Notes",
    collapsible = true,
    notesEndpoint,
    showFormInitially = false,
  } = $props();

  let notes = $state(untrack(() => [...initialNotes]));
  let showNoteForm = $state(untrack(() => showFormInitially));
  let isLoading = $state(false);
  let searchTerm = $state("");

  function authorLabel(note: ClientNote) {
    const name = [note.author.surname?.trim(), note.author.given_name?.trim()]
      .filter(Boolean)
      .join(", ");
    return name || note.author.email?.trim() || "Unknown";
  }

  function searchText(note: ClientNote) {
    return [
      note.note,
      authorLabel(note),
      note.author.email,
      note.job?.number,
      note.job?.description,
    ]
      .filter(Boolean)
      .join(" ");
  }

  const hasMatches = $derived(
    notes.some((note) => searchText(note).toLowerCase().includes(searchTerm.toLowerCase())),
  );

  $effect(() => {
    notes = [...initialNotes];
  });

  function handleNoteCreated(note: ClientNote) {
    notes = [note, ...notes];
    searchTerm = "";
    showNoteForm = false;
  }

  async function refreshNotes() {
    const endpoint =
      notesEndpoint && notesEndpoint.trim().length > 0
        ? notesEndpoint
        : clientId
          ? `/api/clients/${clientId}/notes`
          : "";
    if (!endpoint) {
      return;
    }
    try {
      isLoading = true;
      const refreshed = (await pb.send(endpoint, {
        method: "GET",
      })) as ClientNote[];
      notes = refreshed;
    } catch (error: unknown) {
      globalStore.addError(`error refreshing notes: ${String(error)}`);
    } finally {
      isLoading = false;
    }
  }
</script>

{#snippet actions()}
  <div class="flex items-center gap-2">
    <DsActionButton
      icon={showNoteForm ? "mdi:minus" : "mdi:plus"}
      title={showNoteForm ? "Hide note form" : "Add note"}
      color="green"
      transparentBackground={true}
      action={() => (showNoteForm = !showNoteForm)}
    />
    <DsActionButton
      icon="mdi:refresh"
      title="Refresh notes"
      color="neutral"
      transparentBackground={true}
      action={refreshNotes}
      disabled={isLoading}
    />
  </div>
{/snippet}

{#snippet content()}
  <div class="space-y-4">
    {#if showNoteForm}
      <NoteForm {clientId} jobs={jobOptions} {preselectedJobId} onCreated={handleNoteCreated} />
    {/if}
    <DSList items={notes} search bind:searchTerm {searchText} searchPlaceholder="Filter notes">
      {#snippet headline(note)}{authorLabel(note)}{/snippet}
      {#snippet byline(note)}<span class="text-sm text-neutral-500"
          >{formatDateTime(note.created)}</span
        >{/snippet}
      {#snippet line1(note)}<p class="whitespace-pre-line text-neutral-700">
          {note.note}
        </p>{/snippet}
      {#snippet line2(note)}
        {#if note.job?.id && note.job?.number}
          <a class="text-sm text-blue-600 hover:underline" href={`/jobs/${note.job.id}/details`}
            >{note.job.number}</a
          >
        {/if}
      {/snippet}
    </DSList>
    {#if !hasMatches}
      <p class="p-2 text-sm text-neutral-600 italic" role="status">
        {notes.length ? "No notes match this filter." : "No notes yet."}
      </p>
    {/if}
  </div>
{/snippet}

{#if collapsible}
  <DSCollapsible title={heading} collapsed>
    {#snippet headerActions(isCollapsed)}{#if !isCollapsed}{@render actions()}{/if}{/snippet}
    {#snippet children()}{@render content()}{/snippet}
  </DSCollapsible>
{:else}
  <section aria-label={heading} class="space-y-2">
    <div class="flex items-center justify-between gap-2">
      <h2 class="font-semibold">{heading} ({notes.length})</h2>
      {@render actions()}
    </div>
    {@render content()}
  </section>
{/if}
