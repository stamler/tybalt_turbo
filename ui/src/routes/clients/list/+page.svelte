<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import { globalStore } from "$lib/stores/global";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { pb } from "$lib/pocketbase";
</script>

<!-- Show the list of items here -->
{#if $globalStore.clientsIndex !== null}
  <DsSearchList
    index={$globalStore.clientsIndex}
    inListHeader="Clients"
    fieldName="client"
    uiName="search clients..."
    collectionName="clients"
  >
    {#snippet headline({ name })}{name}{/snippet}
    {#snippet line1({ expand })}
      {#if expand?.client_contacts_via_client}
        <span class="opacity-30">contacts</span>
        {#each expand.client_contacts_via_client as contact}
          <a
            href="mailto:{contact.email}"
            class="rounded-md p-1 hover:cursor-pointer hover:bg-neutral-300"
            title={contact.email}
          >
            {contact.given_name}
            {contact.surname}
          </a>
        {/each}
      {/if}
    {/snippet}
    {#snippet actions({ id })}
      <DsActionButton
        action={`/clients/${id}/edit`}
        icon="mdi:edit-outline"
        title="Edit"
        color="blue"
      />
      <DsActionButton
        action={`/clients/${id}/absorb`}
        icon="mdi:merge"
        title="Absorb other clients into this one"
        color="yellow"
      />
      <DsActionButton
        action={() => pb.collection("clients").delete(id)}
        icon="mdi:delete"
        title="Delete"
        color="red"
      />
    {/snippet}
  </DsSearchList>
{/if}
