<script lang="ts">
  import type { PageData } from "./$types";
  import AbsorbEditor from "$lib/components/AbsorbEditor.svelte";
  import type { ClientsResponse } from "$lib/pocketbase-types";
  export let data: PageData;
</script>

<AbsorbEditor collectionName="clients" targetRecordId={data.params.cid}>
  {#snippet recordSnippet(item: ClientsResponse)}
    <div class="rounded bg-neutral-100 p-2">
      <span class="font-bold">{item.name}</span>
    </div>
  {/snippet}
</AbsorbEditor>
