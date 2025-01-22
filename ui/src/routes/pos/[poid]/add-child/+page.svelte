<script lang="ts">
  import type { PageData } from "./$types";
  import PurchaseOrdersEditor from "$lib/components/PurchaseOrdersEditor.svelte";
  let { data }: { data: PageData } = $props();
</script>

<PurchaseOrdersEditor {data} />
