<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import { goto } from "$app/navigation";
  import type { ClientsPageData } from "$lib/svelte-types";
  import DsActionButton from "./DSActionButton.svelte";
  import type { ClientContactsRecord, ClientContactsResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";
  import { clients } from "$lib/stores/clients";
  import { busdevLeads } from "$lib/stores/busdevLeads";
  import DSAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import { untrack } from "svelte";

  let { data }: { data: ClientsPageData } = $props();

  let errors = $state({} as any);
  let item = $state(untrack(() => data.item));
  let client_contacts = $state(untrack(() => data.client_contacts || []));

  item.business_development_lead = item.business_development_lead ?? "";

  // Initialize busdevLeads store
  busdevLeads.init();

  interface ClientContactWithTempId extends ClientContactsRecord {
    tempId: string;
  }

  function isClientContactWithTempId(
    contact: ClientContactsResponse | ClientContactWithTempId,
  ): contact is ClientContactWithTempId {
    return "tempId" in contact;
  }

  function isClientContactsResponse(
    contact: ClientContactsResponse | ClientContactWithTempId,
  ): contact is ClientContactsResponse {
    return "id" in contact;
  }

  let newContact = $state({
    given_name: "",
    surname: "",
    email: "",
    client: "",
  });
  let newContacts = $state([] as ClientContactWithTempId[]);
  let clientContactsToDelete = $state([] as (ClientContactsResponse | ClientContactWithTempId)[]);

  async function save(event: Event) {
    event.preventDefault();

    // UI validation: business development lead is required
    errors = {};
    if ((item.business_development_lead ?? "").trim() === "") {
      errors = {
        ...errors,
        business_development_lead: { message: "Business development lead is required." },
      } as any;
      return;
    }

    try {
      let clientId = data.id;

      try {
        if (data.editing && clientId !== null) {
          await pb.collection("clients").update(clientId, item);
        } else {
          const createdClient = await pb.collection("clients").create(item);
          clientId = createdClient.id;
        }
      } catch (error: any) {
        const hookErrors = error?.data?.data;
        if (hookErrors !== undefined) {
          errors = hookErrors;
          return;
        }
        globalStore.addError(`error saving client: ${error}`);
        return;
      }

      // Add new client_contacts
      for (const contact of newContacts) {
        try {
          await pb.collection("client_contacts").create({
            ...contact,
            client: clientId,
          });
          // manually reload this client in the clients store so the new
          // contact is visible
          if (clientId !== null) {
            clients.refresh(clientId);
          }
        } catch (error: any) {
          globalStore.addError(
            `error creating contact ${contact.surname}, ${contact.given_name}: ${error}`,
          );
        }
      }

      // Remove deleted client_contacts
      for (const contact of clientContactsToDelete) {
        if (isClientContactWithTempId(contact)) {
          continue;
        }
        try {
          await pb.collection("client_contacts").delete(contact.id);
          // manually reload this client in the clients store so the deleted
          // contact is removed
          if (clientId !== null) {
            clients.refresh(clientId);
          }
        } catch (error: any) {
          globalStore.addError(
            `error deleting contact ${contact.surname}, ${contact.given_name}: ${error}`,
          );
        }
      }
      errors = {};
      goto(`/clients/${clientId}/details`);
    } catch (error: any) {
      errors = error.data.data;
    }
  }

  async function addContact() {
    if (newContact.given_name.trim() === "" || newContact.surname.trim() === "") return;

    newContacts.push({ ...newContact, tempId: Date.now().toString() } as ClientContactWithTempId);
    newContact = {
      given_name: "",
      surname: "",
      email: "",
      client: "",
    };
  }

  async function removeContact(contactId: string) {
    const contact = client_contacts.find((c) => c.id === contactId);
    if (contact !== undefined) {
      // The contact is already in the database, so we need to delete it from
      // the database
      clientContactsToDelete.push(contact);
      client_contacts = client_contacts.filter((contact) => contact.id !== contactId);
    } else {
      // The contact is not in the database, so we need to delete it from
      // the new contacts list
      newContacts = newContacts.filter((contact) => contact.tempId !== contactId);
    }
  }

  function preventDefault(fn: (event: Event) => void) {
    return (event: Event) => {
      event.preventDefault();
      fn(event);
    };
  }
</script>

<form
  class="flex w-full flex-col items-center gap-2 p-2"
  enctype="multipart/form-data"
  onsubmit={save}
>
  <DsTextInput bind:value={item.name as string} {errors} fieldName="name" uiName="Name" />

  {#if $busdevLeads.items.length > 0}
    {#if $busdevLeads.items.length <= 10}
      <DsSelector
        bind:value={item.business_development_lead as string}
        items={[{ id: "", given_name: "", surname: "" }, ...$busdevLeads.items]}
        {errors}
        fieldName="business_development_lead"
        uiName="Business Development Lead"
      >
        {#snippet optionTemplate(option)}
          {#if option.id === ""}
            -- select --
          {:else}
            {option.surname}{option.given_name ? `, ${option.given_name}` : ""}
          {/if}
        {/snippet}
      </DsSelector>
    {:else if $busdevLeads.index !== null}
      <DSAutoComplete
        bind:value={item.business_development_lead as string}
        index={$busdevLeads.index}
        {errors}
        fieldName="business_development_lead"
        uiName="Business Development Lead"
        idField="id"
      >
        {#snippet resultTemplate(option)}
          {option.surname}{option.given_name ? `, ${option.given_name}` : ""}
        {/snippet}
      </DSAutoComplete>
    {/if}
  {:else if $busdevLeads.loading}
    <span class="text-sm text-neutral-500">Loading business development leads…</span>
  {:else}
    <span class="text-sm text-neutral-500">No eligible Business Development Leads found.</span>
  {/if}

  <div
    class="flex w-full flex-col gap-2 {errors.client_contacts !== undefined ? 'bg-red-200' : ''}"
  >
    <label for="client_contacts">Contacts</label>
    <div class="flex flex-col gap-2">
      {#each [...client_contacts, ...newContacts] as contact}
        <div class="flex items-center gap-2 rounded-sm bg-neutral-100 p-2">
          <span>{contact.surname}, {contact.given_name}</span>
          <span>{contact.email}</span>
          <div class="ml-auto flex gap-2">
            {#if isClientContactWithTempId(contact)}
              <button
                class="text-neutral-500"
                onclick={preventDefault(() => removeContact(contact.tempId))}
              >
                &times;
              </button>
            {:else if isClientContactsResponse(contact)}
              <DsActionButton
                action={() =>
                  (window.location.href = `/clients/${data.id}/contacts/${contact.id}/absorb`)}
                icon="mdi:merge"
                title="Absorb other contacts into this one"
                color="yellow"
              />
              <DsActionButton
                action={() => removeContact(contact.id)}
                icon="mdi:delete"
                title="Delete"
                color="red"
              />
            {/if}
          </div>
        </div>
      {/each}
    </div>
    <div class="flex flex-col gap-2 bg-neutral-100 p-2">
      <DsTextInput
        bind:value={newContact.given_name}
        {errors}
        fieldName="newContactGivenName"
        uiName="Given Name"
      />
      <DsTextInput
        bind:value={newContact.surname}
        {errors}
        fieldName="newContactSurname"
        uiName="Surname"
      />
      <DsTextInput
        bind:value={newContact.email}
        {errors}
        fieldName="newContactEmail"
        uiName="Email"
      />
      <DsActionButton
        action={addContact}
        icon="feather:plus-circle"
        color="green"
        title="Add Contact"
      />
    </div>
    {#if errors.client_contacts !== undefined}
      <span class="text-red-600">{errors.client_contacts.message}</span>
    {/if}
  </div>

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton type="submit">Save</DsActionButton>
      <DsActionButton action="/clients/list">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
