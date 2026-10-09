<script lang="ts">
  import { beforeNavigate } from "$app/navigation";
  import { untrack } from "svelte";
  import MiniSearch from "minisearch";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import ContactEditor from "./ContactEditor.svelte";
  import InvoicingInstructions from "./InvoicingInstructions.svelte";
  import { addressText, billingName, contactName, countLabel } from "$lib/clientWorkspace";
  import {
    editableFields,
    invoicingFields,
    workflowError,
    type FieldErrors,
    type WorkflowRecord,
  } from "$lib/clientEditors";

  let {
    client,
    contacts: initialContacts,
    record = {},
    projectContact = "",
    onSaved,
    onCancel,
  }: {
    client: WorkflowRecord;
    contacts: WorkflowRecord[];
    record?: WorkflowRecord;
    projectContact?: string;
    onSaved: (
      record: WorkflowRecord,
      mode?: "create" | "update" | "copy" | "reuse",
    ) => void | Promise<void>;
    onCancel: () => void;
  } = $props();
  // A blank billing_name bills the official name, as does "name".
  let item = $state(
    untrack(() => {
      const fields = editableFields(record, invoicingFields);
      if (fields.billing_name !== "alias") fields.billing_name = "";
      return fields;
    }),
  );
  let saved = $state(untrack(() => JSON.stringify(item)));
  let contacts = $state(untrack(() => [...initialContacts]));
  let contactDraft = $state<WorkflowRecord | null>(null);
  let contactDirty = $state(false);
  let contactBusy = $state(false);
  let savedForNavigation = false;
  let errors = $state<FieldErrors>({});
  let error = $state("");
  let busy = $state(false);
  let completed = $state<{
    record: WorkflowRecord;
    mode: "create" | "update" | "copy" | "reuse";
    unchanged: boolean;
  } | null>(null);
  let decision = $state<{ jobs: number } | null>(null);
  let dialog: HTMLDialogElement;
  const helpId = $props.id();
  const explanations = {
    choose: "Choose how to save your changes.",
    create: "Create a profile with this contact and these instructions.",
    update: "Apply these changes everywhere this profile is used.",
    copy: "Keep the original profile unchanged and create a new profile from these fields.",
    cancel: "Return to editing without saving. Your changes stay in the form.",
    leave: "Leave this editor without saving these changes.",
  };
  // A new profile has one way to save; an existing one offers a choice.
  const defaultHelp = untrack(() => (record.id ? explanations.choose : explanations.create));
  let actionHelp = $state(defaultHelp);
  // Show the explanation for the action under the pointer or keyboard focus.
  const explain = (text: string) => () => (actionHelp = text);
  $effect(() => {
    if (!dialog) return;
    if (decision && !dialog.open) dialog.showModal();
    else if (!decision && dialog.open) dialog.close();
  });
  function cancelDecision() {
    if (busy) return;
    decision = null;
    actionHelp = defaultHelp;
  }
  const canEdit = $derived($globalStore.claims.includes("job"));
  const matchingProfiles = $derived(
    ((client.invoicing_profiles || []) as (WorkflowRecord & { id: string })[]).filter(
      (profile) =>
        profile.client === client.id &&
        profile.contact === item.contact &&
        profile.id !== record.id,
    ),
  );
  const dirty = $derived(JSON.stringify(item) !== saved || contactDirty);
  const recipient = $derived(contacts.find((contact) => contact.id === item.contact));
  const clientAlias = $derived(String(client.alias || "").trim());
  const contactIndex = $derived.by(() => {
    const index = new MiniSearch<WorkflowRecord>({
      fields: ["given_name", "surname", "email", "phone"],
      storeFields: ["given_name", "surname", "email"],
      searchOptions: { combineWith: "AND" },
    });
    index.addAll(contacts);
    return index;
  });
  beforeNavigate(({ cancel }) => {
    if (contactBusy || decision || (busy && !savedForNavigation)) {
      cancel();
      return;
    }
    if (dirty && !window.confirm("Discard unsaved invoicing changes?")) cancel();
  });
  function cancelEdit() {
    if (dirty && !window.confirm("Discard unsaved invoicing changes?")) return;
    saved = JSON.stringify(item);
    contactDirty = false;
    onCancel();
  }
  function contactSaved(contact: WorkflowRecord) {
    contacts = [...contacts.filter((row) => row.id !== contact.id), contact];
    item.contact = contact.id;
    contactDirty = false;
    contactDraft = null;
  }
  function validate() {
    errors = {};
    if (!contacts.some((contact) => contact.id === item.contact))
      errors.contact = { message: "Select a contact for this client." };
    if (item.name.length > 120) errors.name = { message: "Use 120 characters or fewer." };
    return Object.keys(errors).length === 0;
  }
  async function returnToWork() {
    if (!completed) return;
    error = "";
    try {
      await onSaved(completed.record, completed.mode);
    } catch {
      error =
        "Your profile is ready, but the return page could not open. Retry the return; this will not save again.";
    }
  }
  async function retryReturn() {
    if (busy) return;
    busy = true;
    try {
      await returnToWork();
    } finally {
      busy = false;
    }
  }
  async function finish(
    result: WorkflowRecord,
    mode: "create" | "update" | "copy" | "reuse",
    unchanged = false,
  ) {
    saved = JSON.stringify(item);
    decision = null;
    completed = { record: result, mode, unchanged };
    savedForNavigation = true;
    await returnToWork();
  }
  async function reuseProfile(profile: WorkflowRecord) {
    if (
      busy ||
      completed ||
      record.id ||
      !canEdit ||
      !matchingProfiles.some((row) => row.id === profile.id)
    )
      return;
    if (
      [item.name, item.billing_name, item.fax, item.invoicing_instructions].some((value) =>
        String(value).trim(),
      ) &&
      !window.confirm("Use this existing profile and discard the unsaved profile fields?")
    )
      return;
    busy = true;
    error = "";
    try {
      await finish(profile, "reuse");
    } finally {
      busy = false;
    }
  }
  async function persist(mode: "create" | "update" | "copy") {
    const body = { ...editableFields(item, invoicingFields), client: client.id };
    const result =
      mode === "update"
        ? await pb.collection("client_invoicing_information").update(record.id, body)
        : await pb.collection("client_invoicing_information").create(body);
    await finish(result, mode);
  }
  async function save(event?: Event, copy = false) {
    event?.preventDefault();
    if (busy || completed || decision || !canEdit || !validate()) return;
    busy = true;
    savedForNavigation = false;
    error = "";
    try {
      if (record.id && !copy && JSON.stringify(item) === saved) {
        await finish(record, "update", true);
        return;
      }
      if (record.id && !copy) {
        // Refresh usage before confirmation; jobs may have linked this profile since it opened.
        const current = await pb.send<WorkflowRecord>(`/api/clients/${client.id}`, {
          method: "GET",
          requestKey: null,
        });
        const profile = current.invoicing_profiles?.find(
          (profile: WorkflowRecord) => profile.id === record.id,
        );
        if (!profile)
          throw new Error("This invoicing profile is no longer available. Reload the page.");
        const jobs = Number(profile.job_count || 0);
        if (jobs > 1) {
          decision = { jobs };
          actionHelp = explanations.choose;
          return;
        }
      }
      await persist(record.id ? (copy ? "copy" : "update") : "create");
    } catch (e: any) {
      errors = e?.data?.data || {};
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
  async function saveDecision(mode: "update" | "copy") {
    if (busy || !decision || !canEdit) return;
    busy = true;
    error = "";
    try {
      await persist(mode);
    } catch (e: any) {
      errors = e?.data?.data || {};
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
</script>

{#if completed}
  <div class="space-y-3">
    <p role="status">
      {completed.mode === "reuse"
        ? "Using the existing invoicing profile."
        : completed.unchanged
          ? "No changes to save."
          : "Invoicing profile saved."}
    </p>
    {#if error}<p role="alert" class="text-red-700">{error}</p>{/if}
    <DsActionButton action={retryReturn} disabled={busy}>Return to your work</DsActionButton>
  </div>
{:else if contactDraft}
  <p class="mb-4 text-sm text-neutral-600">Save the contact to return to your invoicing profile.</p>
  {#key contactDraft.id || "new"}
    <ContactEditor
      clientId={client.id}
      record={contactDraft}
      guardNavigation={false}
      bind:dirty={contactDirty}
      bind:busy={contactBusy}
      onSaved={contactSaved}
      onCancel={() => {
        contactDraft = null;
        contactDirty = false;
      }}
    />
  {/key}
{:else}
  <form class="flex min-w-0 flex-col gap-4 [&_input]:min-w-0 [&_select]:min-w-0" onsubmit={save}>
    <h2 class="text-xl font-bold">
      {record.id ? "Edit invoicing profile" : "Add invoicing profile"}
    </h2>
    {#if record.id}<p class="text-sm text-neutral-600">
        Used by {countLabel(record.job_count || 0, "job")}. Changes apply wherever this profile is
        used.
      </p>{/if}
    {#if error}<p role="alert" class="text-red-700">{error}</p>{/if}
    <fieldset disabled={busy || !!decision || !canEdit} class="flex min-w-0 flex-col gap-3">
      <DsTextInput
        bind:value={item.name}
        {errors}
        fieldName="name"
        uiName="Profile name (optional)"
        placeholder="For example, Capital projects"
      />
      {#if contacts.length > 10}
        <DsAutoComplete
          bind:value={item.contact}
          index={contactIndex}
          {errors}
          fieldName="contact"
          uiName="Send invoices to"
        >
          {#snippet resultTemplate(row)}{contactName(row as WorkflowRecord)} — {row.email}{/snippet}
        </DsAutoComplete>
      {:else}
        <DsSelector
          bind:value={item.contact}
          items={[
            { id: "", label: "Select a contact" },
            ...contacts.map((contact) => ({
              id: String(contact.id),
              label: `${contactName(contact)} — ${contact.email}`,
            })),
          ]}
          {errors}
          fieldName="contact"
          uiName="Send invoices to"
        >
          {#snippet optionTemplate(row)}{row.label}{/snippet}
        </DsSelector>
      {/if}
      <div class="flex flex-wrap gap-2">
        {#if canEdit}
          <DsActionButton
            action={() => {
              contactDraft = {};
            }}>Add contact</DsActionButton
          >
          {#if recipient}<DsActionButton
              action={() => {
                contactDraft = recipient;
              }}>Edit contact</DsActionButton
            >{/if}
        {/if}
        {#if projectContact && contacts.some((contact) => contact.id === projectContact)}
          <DsActionButton
            action={() => {
              item.contact = projectContact;
            }}>Same as project contact</DsActionButton
          >
        {/if}
      </div>
      {#if !record.id && recipient && matchingProfiles.length}
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions (Enter in profile search must not save a new profile.) -->
        <section
          aria-label="Existing profiles for this contact"
          onkeydown={(event) => {
            if (event.key === "Enter" && event.target instanceof HTMLInputElement)
              event.preventDefault();
          }}
          class="space-y-2 rounded-sm border border-neutral-300 p-3"
        >
          <h3 class="font-semibold">Existing profiles for this contact</h3>
          <p class="text-sm text-neutral-600">
            Use an existing profile, or enter different instructions below to create another.
          </p>
          {#key item.contact}
            <DsList
              items={matchingProfiles}
              search={matchingProfiles.length > 10}
              pagination={matchingProfiles.length > 10}
              perPage={10}
              searchPlaceholder="Find an existing profile"
              searchText={(profile) =>
                [profile.name, profile.invoicing_instructions, profile.fax]
                  .filter(Boolean)
                  .join(" ")}
            >
              {#snippet headline(profile)}{profile.name || contactName(recipient)}{/snippet}
              {#snippet line1(profile)}
                <InvoicingInstructions text={profile.invoicing_instructions || ""} />
                {#if profile.fax}<p class="text-sm">Fax: {profile.fax}</p>{/if}
              {/snippet}
              {#snippet actions(profile)}<DsActionButton
                  action={() => reuseProfile(profile)}
                  disabled={busy || !canEdit}>Use this profile</DsActionButton
                >{/snippet}
            </DsList>
          {/key}
        </section>
      {/if}
      {#if clientAlias || item.billing_name === "alias"}
        <fieldset class="flex flex-col gap-1">
          <legend class="mb-1">Name on PAs and invoices</legend>
          <label class="flex items-center gap-2">
            <input type="radio" name="billing_name" value="" bind:group={item.billing_name} />
            {client.name}
            <span class="text-sm text-neutral-600">(name)</span>
          </label>
          <label class="flex items-center gap-2">
            <input
              type="radio"
              name="billing_name"
              value="alias"
              bind:group={item.billing_name}
              disabled={!clientAlias}
            />
            {clientAlias || "No alias"}
            <span class="text-sm text-neutral-600">(alias)</span>
          </label>
          {#if errors.billing_name}<p class="text-red-600">{errors.billing_name.message}</p>{/if}
        </fieldset>
      {:else}
        <p class="text-sm text-neutral-600">
          PAs and invoices use the client name. To use a different name, add an alias to the client.
        </p>
      {/if}
      {#if recipient}
        <div class="rounded-sm bg-neutral-100 p-3 text-sm">
          <p class="font-semibold">Invoice address</p>
          <p>{billingName(client as { name: string; alias?: string }, item)}</p>
          <p class="whitespace-pre-line">
            {addressText(recipient) || addressText(client) || "No postal address entered"}
          </p>
          {#if !addressText(recipient)}<p class="text-neutral-600">Uses the client address.</p>{/if}
        </div>
      {/if}
      <label class="flex flex-col gap-2" for="invoicing-instructions"
        >Invoice instructions (optional)
        <textarea
          id="invoicing-instructions"
          name="invoicing_instructions"
          class="rounded border border-neutral-300 p-2"
          rows="4"
          bind:value={item.invoicing_instructions}></textarea>
      </label>
      {#if errors.invoicing_instructions}<p class="text-red-600">
          {errors.invoicing_instructions.message}
        </p>{/if}
      <DsTextInput bind:value={item.fax} {errors} fieldName="fax" uiName="Fax (optional)" />
    </fieldset>
    <div class="flex flex-wrap gap-2">
      <button
        type="submit"
        disabled={busy || !!decision || !canEdit}
        aria-describedby={helpId + "-form"}
        onmouseenter={explain(record.id ? explanations.update : explanations.create)}
        onfocus={explain(record.id ? explanations.update : explanations.create)}
        class="rounded-xs bg-yellow-200 px-2 py-1 hover:bg-yellow-300 disabled:opacity-40"
        >Save invoicing profile</button
      >
      {#if record.id && canEdit}
        <button
          type="button"
          disabled={busy || !!decision || !canEdit}
          onclick={() => save(undefined, true)}
          aria-describedby={helpId + "-form"}
          onmouseenter={explain(explanations.copy)}
          onfocus={explain(explanations.copy)}
          class="rounded-xs bg-neutral-200 px-2 py-1 hover:bg-neutral-300 disabled:opacity-40"
          >Save as new profile</button
        >
      {/if}
      <button
        type="button"
        onclick={cancelEdit}
        disabled={busy || !!decision}
        aria-describedby={helpId + "-form"}
        onmouseenter={explain(explanations.leave)}
        onfocus={explain(explanations.leave)}
        class="rounded-xs bg-neutral-200 px-2 py-1 hover:bg-neutral-300 disabled:opacity-40"
        >Cancel</button
      >
    </div>
    <p id={helpId + "-form"} class="min-h-10 text-sm text-neutral-600" aria-live="polite">
      {actionHelp}
    </p>
  </form>
{/if}

<dialog
  bind:this={dialog}
  oncancel={(event) => {
    event.preventDefault();
    cancelDecision();
  }}
  aria-labelledby={helpId + "-title"}
  aria-describedby={helpId + "-usage"}
  class="m-auto w-[32rem] max-w-[calc(100vw-2rem)] rounded-md border border-neutral-300 bg-white p-5 text-neutral-800 shadow-lg backdrop:bg-black/40"
>
  {#if decision}
    <h2 id={helpId + "-title"} class="mb-3 text-xl font-semibold">Save invoicing profile</h2>
    <p id={helpId + "-usage"}>
      This profile is used by {decision.jobs} jobs.
    </p>
    <p class="mt-2">Update the shared profile, or save your changes as a new profile?</p>
    {#if error}<p role="alert" class="mt-2 text-red-700">{error}</p>{/if}
    <div class="mt-4 flex flex-wrap gap-2">
      <button
        type="button"
        disabled={busy}
        onclick={() => saveDecision("update")}
        aria-describedby={helpId + "-choice"}
        onmouseenter={explain(explanations.update)}
        onfocus={explain(explanations.update)}
        class="rounded-xs bg-yellow-200 px-2 py-1 hover:bg-yellow-300 disabled:opacity-40"
        >Update shared profile</button
      >
      <button
        type="button"
        disabled={busy}
        onclick={() => saveDecision("copy")}
        aria-describedby={helpId + "-choice"}
        onmouseenter={explain(explanations.copy)}
        onfocus={explain(explanations.copy)}
        class="rounded-xs bg-neutral-200 px-2 py-1 hover:bg-neutral-300 disabled:opacity-40"
        >Save as new profile</button
      >
      <button
        type="button"
        disabled={busy}
        onclick={cancelDecision}
        aria-describedby={helpId + "-choice"}
        onmouseenter={explain(explanations.cancel)}
        onfocus={explain(explanations.cancel)}
        class="rounded-xs bg-neutral-200 px-2 py-1 hover:bg-neutral-300 disabled:opacity-40"
        >Cancel</button
      >
    </div>
    <p id={helpId + "-choice"} class="mt-3 min-h-10 text-sm text-neutral-600" aria-live="polite">
      {actionHelp}
    </p>
  {/if}
</dialog>
