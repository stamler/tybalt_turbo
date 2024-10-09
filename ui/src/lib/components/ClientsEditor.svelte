<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import { goto } from "$app/navigation";
  import type { ClientsPageData } from "$lib/svelte-types";
  import DsActionButton from "./DSActionButton.svelte";
  import type { ContactsRecord, ContactsResponse } from "$lib/pocketbase-types";

  let { data }: { data: ClientsPageData } = $props();

  let errors = $state({} as any);
  let item = $state(data.item);
  let contacts = $state(data.contacts || []);

  let newContact = $state({
    given_name: "",
    surname: "",
    email: "",
  } as ContactsRecord);
  let newContacts = $state([] as ContactsResponse[]);
  let contactsToDelete = $state([] as string[]);

  async function save(event: Event) {
    event.preventDefault();

    try {
      let clientId = data.id;

      if (data.editing && clientId !== null) {
        await pb.collection("clients").update(clientId, item);
      } else {
        const createdClient = await pb.collection("clients").create(item);
        clientId = createdClient.id;
      }

      // Add new contacts
      for (const contact of newContacts) {
        await pb.collection("contacts").create(
          {
            ...contact,
            client: clientId,
          },
          { returnRecord: true },
        );
      }

      // Remove deleted contacts
      for (const contactId of contactsToDelete) {
        await pb.collection("contacts").delete(contactId);
      }

      errors = {};
      goto("/clients/list");
    } catch (error: any) {
      errors = error.data.data;
    }
  }

  async function addContact() {
    if (newContact.given_name.trim() === "" || newContact.surname.trim() === "") return;

    newContacts.push({ ...newContact, id: Date.now().toString() } as ContactsResponse);
    newContact = {
      given_name: "",
      surname: "",
      email: "",
      client: "",
    };
  }

  async function removeContact(contactId: string) {
    if (contacts.find((c) => c.id === contactId)) {
      contactsToDelete.push(contactId);
      contacts = contacts.filter((contact) => contact.id !== contactId);
    } else {
      newContacts = newContacts.filter((contact) => contact.id !== contactId);
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
      {#each [...contacts, ...newContacts] as contact (contact.id)}
        <div class="flex items-center gap-2 rounded bg-neutral-100 p-2">
          <span>{contact.surname}, {contact.given_name}</span>
          <span>{contact.email}</span>
          <button
            class="ml-auto text-neutral-500"
            onclick={preventDefault(() => removeContact(contact.id))}
          >
            &times;
          </button>
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
