<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import type { ClientNote } from "../../routes/clients/[cid]/details/+page";

  type NoteJobOption = {
    id: string;
    number: string;
    description: string;
  };

  let {
    clientId,
    jobs = [] as NoteJobOption[],
    onCreated,
  }: {
    clientId: string;
    jobs?: NoteJobOption[];
    onCreated?: (note: ClientNote) => void;
  } = $props();

  let note = $state("");
  let selectedJob = $state("");
  let jobNotApplicable = $state(false);
  let errors = $state({} as Record<string, { message: string }>);
  let saving = $state(false);

  const jobOptions = $derived([{ id: "", number: "", description: "-- Select job --" }, ...jobs]);

  function resetForm() {
    note = "";
    selectedJob = "";
    jobNotApplicable = false;
    errors = {};
  }

  function validateLocally() {
    const nextErrors: Record<string, { message: string }> = {};
    const trimmedNote = note.trim();
    if (trimmedNote.length === 0) {
      nextErrors.note = { message: "Note is required" };
    }
    if (note.length > 1000) {
      nextErrors.note = { message: "Note must be 1000 characters or less" };
    }
    if (!jobNotApplicable && selectedJob.trim() === "") {
      nextErrors.job = { message: "Select a job or mark not applicable" };
    }
    errors = nextErrors;
    return Object.keys(nextErrors).length === 0;
  }

  async function submit(event: Event) {
    event.preventDefault();
    if (!validateLocally()) {
      return;
    }

    try {
      saving = true;
      const payload = {
        client: clientId,
        note: note.trim(),
        job: jobNotApplicable ? "" : selectedJob,
        job_not_applicable: jobNotApplicable,
      };
      const created = await pb.collection("client_notes").create(payload);
      const refreshedNotes = (await pb.send(`/api/clients/${clientId}/notes`, {
        method: "GET",
      })) as ClientNote[];
      const latest = refreshedNotes.find((n) => n.id === created.id) ?? refreshedNotes[0];
      if (latest) {
        onCreated?.(latest);
      }
      resetForm();
    } catch (error: unknown) {
      const pocketError = error as
        | { data?: { data?: Record<string, { message: string }> } }
        | undefined;
      const hookErrors = pocketError?.data?.data;
      if (hookErrors) {
        errors = hookErrors;
        return;
      }
      globalStore.addError(`error creating note: ${String(error)}`);
    } finally {
      saving = false;
    }
  }
</script>

<form class="flex flex-col gap-3" onsubmit={submit}>
  <label class="flex flex-col gap-1">
    <span class="text-sm font-semibold">Note</span>
    <textarea
      bind:value={note}
      maxlength={1000}
      rows={3}
      class="rounded border border-neutral-300 p-2"
    ></textarea>
    <span class="text-xs text-neutral-500">{note.length}/1000</span>
    {#if errors.note}
      <span class="text-sm text-red-600">{errors.note.message}</span>
    {/if}
  </label>

  <div class="space-y-2">
    <DsSelector
      bind:value={selectedJob}
      items={jobOptions}
      {errors}
      fieldName="job"
      uiName="Related Job"
      disabled={jobNotApplicable}
    >
      {#snippet optionTemplate(option)}
        {option.number ? `${option.number} — ${option.description}` : option.description}
      {/snippet}
    </DsSelector>

    <label class="flex items-center gap-2 text-sm">
      <input
        type="checkbox"
        bind:checked={jobNotApplicable}
        onchange={() => {
          if (jobNotApplicable) selectedJob = "";
        }}
      />
      Not tied to a specific job
    </label>
    {#if errors.job_not_applicable}
      <span class="text-sm text-red-600">{errors.job_not_applicable.message}</span>
    {/if}
    {#if errors.job}
      <span class="text-sm text-red-600">{errors.job.message}</span>
    {/if}
  </div>

  <div class="flex gap-2">
    <DsActionButton type="submit" disabled={saving}>Add Note</DsActionButton>
    <DsActionButton action={resetForm} type="button" color="neutral" disabled={saving}
      >Clear</DsActionButton
    >
  </div>
</form>
