<script lang="ts">
  import AbsorbEditor from "$lib/components/AbsorbEditor.svelte";
  import type { ClientsResponse } from "$lib/pocketbase-types";
  import { clients } from "$lib/stores/clients";
  import { clientLabel } from "$lib/clientWorkspace";
  import { page } from "$app/stores";
</script>

<AbsorbEditor
  collectionName="clients"
  targetRecordId={$page.params.cid ?? ""}
  availableRecords={$clients.items}
  autoCompleteIndex={$clients.index as unknown as any}
>
  {#snippet recordSnippet(item: ClientsResponse)}
    {clientLabel(item)}
  {/snippet}
</AbsorbEditor>
