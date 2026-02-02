<script lang="ts">
  import DSCollapsible from "$lib/components/DSCollapsible.svelte";
  import ClientNoteItem from "$lib/components/ClientNoteItem.svelte";
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
    notesEndpoint,
    showFormInitially = false,
  } = $props();

  let notes = $state(untrack(() => [...initialNotes]));
  let showNoteForm = $state(untrack(() => showFormInitially));
  let isLoading = $state(false);

  $effect(() => {
    notes = [...initialNotes];
  });

  function handleNoteCreated(note: ClientNote) {
    notes = [note, ...notes];
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

<DSCollapsible title={heading} collapsed>
  {#snippet headerActions(isCollapsed)}
    {#if !isCollapsed}
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
    {/if}
  {/snippet}

  {#snippet children()}
    <div class="mt-2 space-y-4">
      {#if showNoteForm}
        <NoteForm {clientId} jobs={jobOptions} {preselectedJobId} onCreated={handleNoteCreated} />
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
