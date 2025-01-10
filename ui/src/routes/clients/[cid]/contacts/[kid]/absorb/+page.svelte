<script lang="ts">
  import type { PageData } from "./$types";
  import AbsorbEditor from "$lib/components/AbsorbEditor.svelte";
  import type { ClientContactsResponse } from "$lib/pocketbase-types";
  export let data: PageData;
</script>

<AbsorbEditor collectionName="client_contacts" targetRecordId={data.params.kid}>
  {#snippet recordSnippet(item: ClientContactsResponse)}
    <div class="rounded bg-neutral-100 p-2">
      <a
        href="mailto:{item.email}"
        class="rounded-md p-1 hover:cursor-pointer hover:bg-neutral-300"
        title={item.email}
      >
        {item.given_name}
        {item.surname}
      </a>
    </div>
  {/snippet}
</AbsorbEditor>
