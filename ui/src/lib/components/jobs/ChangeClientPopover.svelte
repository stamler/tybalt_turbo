<script lang="ts">
  import DSPopover from "$lib/components/DSPopover.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { clients } from "$lib/stores/clients";
  import { pb } from "$lib/pocketbase";
  import { resolve } from "$app/paths";
  import { ClientResponseError } from "pocketbase";
  import { clientLabel } from "$lib/clientWorkspace";
  import { formatContactName, invoicingProfileLabel } from "$lib/utilities";
  import type {
    ClientContactsResponse,
    ClientInvoicingInformationResponse,
  } from "$lib/pocketbase-types";

  // The reason limits match the server's, which keep the client notes that
  // quote the reason within their length limit.
  const REASON_MIN = 20;
  const REASON_MAX = 500;

  let {
    show = $bindable(),
    job,
    onchanged,
  }: {
    show: boolean;
    job: {
      id: string;
      number: string;
      client: { id: string; name?: string; alias?: string };
      children?: { id: string; number: string }[];
    };
    onchanged: () => void | Promise<void>;
  } = $props();

  // The parent mounts this dialog only while it is open, so each open starts a
  // fresh form and the client list loads only when it is needed.
  clients.init();

  let client = $state("");
  let contact = $state("");
  let invoicingInformation = $state("");
  let reason = $state("");
  let contacts = $state<ClientContactsResponse[]>([]);
  let profiles = $state<ClientInvoicingInformationResponse[]>([]);
  let loading = $state(false);
  let loadError = $state(false);
  let submitting = $state(false);
  let errors = $state({} as Record<string, { message: string }>);
  let globalError = $state<string | null>(null);
  let loadId = 0;

  const children = $derived(job.children ?? []);
  const reasonLength = $derived(reason.trim().length);
  const newClientName = $derived(
    client ? clientLabel($clients.items.find((row) => row.id === client) ?? {}) : "",
  );

  // A contact and profile belong to one client, so choosing a client reloads
  // both lists and clears the earlier choices.
  $effect(() => {
    const selected = client;
    contact = "";
    invoicingInformation = "";
    load(selected);
  });

  async function load(selected: string) {
    const id = ++loadId;
    contacts = [];
    profiles = [];
    loadError = false;
    loading = Boolean(selected);
    if (!selected) return;
    try {
      const filter = pb.filter("client={:client}", { client: selected });
      const [contactRows, profileRows] = await Promise.all([
        pb
          .collection("client_contacts")
          .getFullList({ filter, sort: "surname,given_name", requestKey: null }),
        pb.collection("client_invoicing_information").getFullList({ filter, requestKey: null }),
      ]);
      if (id !== loadId) return;
      contacts = contactRows;
      profiles = profileRows;
    } catch {
      if (id === loadId) loadError = true;
    } finally {
      if (id === loadId) loading = false;
    }
  }

  function close() {
    if (submitting) return;
    show = false;
  }

  async function submit() {
    if (submitting) return;
    submitting = true;
    errors = {};
    globalError = null;
    try {
      await pb.send(`/api/jobs/${job.id}/change-client`, {
        method: "POST",
        body: {
          client,
          contact,
          invoicing_information: invoicingInformation,
          reason,
        },
      });
      show = false;
      await onchanged();
    } catch (error: unknown) {
      const response = error instanceof ClientResponseError ? error.data : {};
      errors = (response?.data as Record<string, { message: string }> | undefined) ?? {};
      globalError = response?.message ?? "The client could not be changed. Try again.";
    } finally {
      submitting = false;
    }
  }
</script>

<DSPopover
  bind:show
  title="Change Client"
  subtitle={`Move project ${job.number} from ${clientLabel(job.client)} to another client. Use this when the client was chosen in error or the work now belongs to another client.`}
  error={globalError}
  {submitting}
  submitLabel="Change Client"
  onSubmit={submit}
  onCancel={close}
>
  {#if $clients.index !== null}
    <DsAutoComplete
      bind:value={client}
      index={$clients.index}
      {errors}
      fieldName="client"
      uiName="New client"
      excludeIds={[job.client.id]}
      disabled={submitting}
    >
      {#snippet resultTemplate(row)}{clientLabel({ name: row.name, alias: row.alias })}{/snippet}
    </DsAutoComplete>
  {:else}
    <p role="status" class="text-sm text-neutral-600">Loading clients…</p>
  {/if}

  {#if client}
    {#if loading}
      <p role="status" class="text-sm text-neutral-600">Loading contacts and invoicing profiles…</p>
    {:else if loadError}
      <p role="alert" class="text-sm text-red-700">
        Could not load this client's contacts and invoicing profiles.
        <button type="button" class="ml-1 text-blue-700 underline" onclick={() => load(client)}
          >Try again</button
        >
      </p>
    {:else if !contacts.length || !profiles.length}
      <div class="rounded-sm border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
        <p>
          {newClientName || "This client"} needs
          {!contacts.length ? "a contact and an invoicing profile" : "an invoicing profile"} before a
          project can move to it.
        </p>
        <p class="mt-1">
          <a
            href={resolve("/clients/[cid]/details", { cid: client })}
            target="_blank"
            rel="noopener"
            class="text-blue-700 underline">Open the client in a new tab</a
          >
          to add {!contacts.length ? "them" : "one"}, then
          <button type="button" class="text-blue-700 underline" onclick={() => load(client)}
            >check again</button
          >.
        </p>
      </div>
    {:else}
      <label class="flex flex-col gap-1 {errors.contact ? 'bg-red-200' : ''}">
        <span>Client contact</span>
        <select
          bind:value={contact}
          class="rounded-sm border border-neutral-300 px-1"
          disabled={submitting}
        >
          <option value="">Select a contact</option>
          {#each contacts as row (row.id)}
            <option value={row.id}>{formatContactName(row)}</option>
          {/each}
        </select>
        {#if errors.contact}<span class="text-sm text-red-600">{errors.contact.message}</span>{/if}
      </label>
      <label class="flex flex-col gap-1 {errors.invoicing_information ? 'bg-red-200' : ''}">
        <span>Invoicing profile</span>
        <select
          bind:value={invoicingInformation}
          class="rounded-sm border border-neutral-300 px-1"
          disabled={submitting}
        >
          <option value="">Select invoicing profile</option>
          {#each profiles as row (row.id)}
            <option value={row.id}>{invoicingProfileLabel(row, profiles, contacts)}</option>
          {/each}
        </select>
        {#if errors.invoicing_information}<span class="text-sm text-red-600"
            >{errors.invoicing_information.message}</span
          >{/if}
      </label>
    {/if}
  {/if}

  <label class="flex flex-col gap-1 {errors.reason ? 'bg-red-200' : ''}">
    <span>Reason for the change</span>
    <textarea
      bind:value={reason}
      maxlength={REASON_MAX}
      rows={3}
      class="rounded-sm border border-neutral-300 p-2"
      disabled={submitting}></textarea>
    <span class="text-xs text-neutral-500"
      >{reasonLength < REASON_MIN
        ? `At least ${REASON_MIN} characters (${reasonLength} so far)`
        : `${reason.length}/${REASON_MAX}`}</span
    >
    {#if errors.reason}<span class="text-sm text-red-600">{errors.reason.message}</span>{/if}
  </label>

  <div class="rounded-sm border border-neutral-300 bg-neutral-50 p-3 text-sm text-neutral-700">
    {#if children.length}
      <p>
        Subjob{children.length === 1 ? "" : "s"}
        {children.map((child) => child.number).join(", ")}
        will move too, with the same contact and invoicing profile.
      </p>
    {/if}
    <p class={children.length ? "mt-1" : ""}>
      The reason is saved as a client note under both clients. The project's managers and the people
      who approved its value are emailed. Approved requests and value changes keep the client they
      were approved for.
    </p>
  </div>
</DSPopover>
