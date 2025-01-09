<script lang="ts">
  import type { PageData } from "./$types";
  import AbsorbEditor from "$lib/components/AbsorbEditor.svelte";

  export let data: PageData;
</script>

<AbsorbEditor collectionName="clients" targetRecordId={data.params.cid}>
  {#snippet targetRecordSnippet()}
    <div class="rounded bg-neutral-100 p-2">
      <span class="font-bold">{data.record.name}</span>
    </div>
  {/snippet}
</AbsorbEditor>
