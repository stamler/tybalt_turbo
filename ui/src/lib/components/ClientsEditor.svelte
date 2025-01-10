<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import { goto } from "$app/navigation";
  import type { ClientsPageData } from "$lib/svelte-types";
  import DsActionButton from "./DSActionButton.svelte";
  import type { ClientContactsRecord, ClientContactsResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";

  let { data }: { data: ClientsPageData } = $props();

  let errors = $state({} as any);
  let item = $state(data.item);
  let client_contacts = $state(data.client_contacts || []);

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
  } as ClientContactsRecord);
  let newContacts = $state([] as ClientContactWithTempId[]);
  let clientContactsToDelete = $state([] as (ClientContactsResponse | ClientContactWithTempId)[]);

  async function save(event: Event) {
    event.preventDefault();

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
        globalStore.addError(`error saving client: ${error}`);
      }

      // Add new client_contacts
      for (const contact of newContacts) {
        try {
          await pb.collection("client_contacts").create(
            {
              ...contact,
              client: clientId,
            },
            { returnRecord: true },
          );
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
        } catch (error: any) {
          globalStore.addError(
            `error deleting contact ${contact.surname}, ${contact.given_name}: ${error}`,
          );
        }
      }

      // reload clients in the global store
      globalStore.refresh("clients");

      errors = {};
      goto("/clients/list");
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

  <div
    class="flex w-full flex-col gap-2 {errors.client_contacts !== undefined ? 'bg-red-200' : ''}"
  >
    <label for="client_contacts">Contacts</label>
    <div class="flex flex-col gap-2">
      {#each [...client_contacts, ...newContacts] as contact}
        <div class="flex items-center gap-2 rounded bg-neutral-100 p-2">
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
