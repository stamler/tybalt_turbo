<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import { goto } from "$app/navigation";
  import type { ClientsPageData } from "$lib/svelte-types";
  import DsActionButton from "./DSActionButton.svelte";
  import type { ContactsRecord, ContactsResponse } from "$lib/pocketbase-types";
  import { globalStore } from "$lib/stores/global";

  let { data }: { data: ClientsPageData } = $props();

  let errors = $state({} as any);
  let item = $state(data.item);
  let contacts = $state(data.contacts || []);

  interface ContactWithTempId extends ContactsRecord {
    tempId: string;
  }

  function isContactWithTempId(
    contact: ContactsResponse | ContactWithTempId,
  ): contact is ContactWithTempId {
    return (contact as ContactWithTempId).tempId !== undefined;
  }

  let newContact = $state({
    given_name: "",
    surname: "",
    email: "",
  } as ContactsRecord);
  let newContacts = $state([] as ContactWithTempId[]);
  let contactsToDelete = $state([] as (ContactsResponse | ContactWithTempId)[]);

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

      // Add new contacts
      for (const contact of newContacts) {
        try {
          await pb.collection("contacts").create(
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

      // Remove deleted contacts
      for (const contact of contactsToDelete) {
        if (isContactWithTempId(contact)) {
          continue;
        }
        try {
          await pb.collection("contacts").delete(contact.id);
        } catch (error: any) {
          globalStore.addError(
            `error deleting contact ${contact.surname}, ${contact.given_name}: ${error}`,
          );
        }
      }

      errors = {};
      goto("/clients/list");
    } catch (error: any) {
      errors = error.data.data;
    }
  }

  async function addContact() {
    if (newContact.given_name.trim() === "" || newContact.surname.trim() === "") return;

    newContacts.push({ ...newContact, tempId: Date.now().toString() } as ContactWithTempId);
    newContact = {
      given_name: "",
      surname: "",
      email: "",
      client: "",
    };
  }

  async function removeContact(contactId: string) {
    const contact = contacts.find((c) => c.id === contactId);
    if (contact !== undefined) {
      // The contact is already in the database, so we need to delete it from
      // the database
      contactsToDelete.push(contact);
      contacts = contacts.filter((contact) => contact.id !== contactId);
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

  <div class="flex w-full flex-col gap-2 {errors.contacts !== undefined ? 'bg-red-200' : ''}">
    <label for="contacts">Contacts</label>
    <div class="flex flex-col gap-2">
      {#each [...contacts, ...newContacts] as contact}
        <div class="flex items-center gap-2 rounded bg-neutral-100 p-2">
          <span>{contact.surname}, {contact.given_name}</span>
          <span>{contact.email}</span>
          {#if isContactWithTempId(contact)}
            <button
              class="ml-auto text-neutral-500"
              onclick={preventDefault(() => removeContact(contact.tempId))}
            >
              &times;
            </button>
          {:else}
            <button
              class="ml-auto text-neutral-500"
              onclick={preventDefault(() => removeContact(contact.id))}
            >
              &times;
            </button>
          {/if}
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
    {#if errors.contacts !== undefined}
      <span class="text-red-600">{errors.contacts.message}</span>
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
