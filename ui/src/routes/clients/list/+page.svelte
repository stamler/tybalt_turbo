<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import { globalStore } from "$lib/stores/global";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
</script>

<!-- Show the list of items here -->
<DsList items={$globalStore.clients} search={true}>
  {#snippet headline({ name })}{name}{/snippet}
  {#snippet line1({ expand })}
    <span class="opacity-30">contacts</span>
    {#each expand?.contacts_via_client as contact}
      <a
        href="mailto:{contact.email}"
        class="rounded-md p-1 hover:cursor-pointer hover:bg-neutral-300"
        title={contact.email}
      >
        {contact.given_name}
        {contact.surname}
      </a>
    {/each}
  {/snippet}
  {#snippet actions({ id })}
    <DsActionButton
      action={`/clients/${id}/edit`}
      icon="mdi:edit-outline"
      title="Edit"
      color="blue"
    />
  {/snippet}
</DsList>
