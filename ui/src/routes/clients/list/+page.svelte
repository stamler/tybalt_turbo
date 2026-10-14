<script lang="ts">
  import DsSearchList from "$lib/components/DSSearchList.svelte";
  import { clients, type ClientApiResponse } from "$lib/stores/clients";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { pb } from "$lib/pocketbase";
  import { clientLabel } from "$lib/clientWorkspace";
  import { globalStore } from "$lib/stores/global";
  import { errorFieldCode, errorStatus, saveErrorMessage } from "$lib/utilities";

  // initialize the stores, noop if already initialized
  clients.init();

  async function del(id: string, label: string): Promise<void> {
    const inUse = `${label} could not be deleted because jobs, job requests, invoicing profiles or notes use it. Absorb it into another client instead.`;
    try {
      await pb.collection("clients").delete(id);
      // Remove the row now instead of waiting for the realtime delete event.
      clients.remove(id);
      return;
    } catch (error: unknown) {
      const status = errorStatus(error);
      // The delete hook refuses a client that owns a job or request or has
      // notes. Reload it so its Delete button is hidden.
      if (status === 400 && errorFieldCode(error, "id") === "in_use") {
        await clients.reloadClient(id).catch(() => null);
        globalStore.addError(inUse);
        return;
      }
      if (status !== 404) {
        globalStore.addError(
          `${label} could not be deleted: ${saveErrorMessage(error, "unknown error")}`,
        );
        return;
      }
    }
    // PocketBase answers 404 both when the client is gone and when the delete
    // rule refuses the request: the user lacks the job claim or an active admin
    // profile, or a job, job request or invoicing profile uses the client.
    // Reload the client to tell which.
    const fresh = await clients.reloadClient(id).catch(() => undefined);
    if (fresh === null) return; // already deleted, and now removed from the list
    globalStore.addError(
      fresh?.referenced
        ? inUse
        : `${label} could not be deleted. Only active users with the job claim can delete clients.`,
    );
  }
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
    {#snippet headline({ id, name, alias })}
      <a href={`/clients/${id}/details`} class="text-blue-600 hover:underline"
        >{clientLabel({ name, alias })}</a
      >
    {/snippet}
    {#snippet byline({ referencing_jobs_count })}
      <span class="mr-2">
        {referencing_jobs_count === 0
          ? "no jobs"
          : referencing_jobs_count === 1
            ? "1 job"
            : `${referencing_jobs_count} jobs`}
      </span>
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
    {#snippet actions({ id, name, alias, referenced }: ClientApiResponse)}
      <DsActionButton
        action={`/clients/${id}/details?edit=client`}
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
      <!-- Only users with the job claim can delete clients, and clients that are in use must be absorbed instead. -->
      {#if !referenced && $globalStore.claims.includes("job")}
        <DsActionButton
          action={() => del(id, clientLabel({ name, alias }))}
          icon="mdi:delete"
          title="Delete"
          color="red"
        />
      {/if}
    {/snippet}
  </DsSearchList>
{/if}
