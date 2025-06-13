<script lang="ts">
  import type { PageData } from "./$types";
  import AbsorbEditor from "$lib/components/AbsorbEditor.svelte";
  import type { ClientContactsResponse } from "$lib/pocketbase-types";
  export let data: PageData;
</script>

<AbsorbEditor collectionName="client_contacts" targetRecordId={data.params.kid}>
  {#snippet recordSnippet(item: ClientContactsResponse)}
    {item.given_name} {item.surname} — {item.email}
  {/snippet}
</AbsorbEditor>
