<script lang="ts">
  import ClientWorkspaceShell from "$lib/components/ClientWorkspaceShell.svelte";
  import type { PageData } from "./$types";
  import AbsorbEditor from "$lib/components/AbsorbEditor.svelte";
  import type { ClientContactsResponse } from "$lib/pocketbase-types";
  import MiniSearch from "minisearch";
  import { page } from "$app/stores";
  const INDEX_THRESHOLD = 30;
  export let data: PageData;

  const availableRecords: ClientContactsResponse[] = data.contacts;
  let autoCompleteIndex: MiniSearch<ClientContactsResponse> | null = null;

  if (availableRecords.length > INDEX_THRESHOLD) {
    autoCompleteIndex = new MiniSearch<ClientContactsResponse>({
      // include "id" to allow searching by id when a result is selected
      fields: ["id", "given_name", "surname", "email"],
      storeFields: ["id", "given_name", "surname", "email"],
      searchOptions: {
        combineWith: "AND",
      },
    });
    autoCompleteIndex.addAll(availableRecords);
  }
</script>

<ClientWorkspaceShell clientId={data.client.id} clientName={data.client.name}>
  <AbsorbEditor
    collectionName="client_contacts"
    returnTo={data.returnTo}
    targetRecordId={$page.params.kid!}
    {availableRecords}
    autoCompleteIndex={autoCompleteIndex as unknown as any}
  >
    {#snippet recordSnippet(item: ClientContactsResponse)}
      {item.given_name} {item.surname} — {item.email}
    {/snippet}
  </AbsorbEditor>
</ClientWorkspaceShell>
