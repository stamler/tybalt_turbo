<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import { clients } from "$lib/stores/clients";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { pb } from "$lib/pocketbase";

  // initialize the stores, noop if already initialized
  clients.init();
</script>

<!-- Show the list of items here -->
{#if $clients.index !== null}
  <DsSearchList
    index={$clients.index}
    inListHeader="Clients"
    fieldName="client"
    uiName="search clients..."
    collectionName="clients"
  >
    {#snippet headline({ id, name })}
      <a href={`/clients/${id}/details`} class="text-blue-600 hover:underline">{name}</a>
    {/snippet}
    {#snippet line1({ contacts })}
      {#if contacts && contacts.length > 0}
        <span class="opacity-30">contacts</span>
        {#each contacts as contact}
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
