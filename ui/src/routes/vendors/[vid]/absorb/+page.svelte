<script lang="ts">
  import AbsorbEditor from "$lib/components/AbsorbEditor.svelte";
  import type { VendorsResponse } from "$lib/pocketbase-types";
  import { vendors } from "$lib/stores/vendors";
  import { page } from "$app/stores";
  import { trimmedOrEmpty } from "$lib/utilities";

  // initialize the stores, noop if already initialized
  vendors.init();
</script>

<AbsorbEditor
  collectionName="vendors"
  targetRecordId={$page.params.vid}
  availableRecords={$vendors.items}
  autoCompleteIndex={$vendors.index as unknown as any}
>
  {#snippet recordSnippet(item: VendorsResponse)}
    {item.name}
    {#if trimmedOrEmpty(item.alias)}
      ({trimmedOrEmpty(item.alias)})
    {/if}
  {/snippet}
</AbsorbEditor>
