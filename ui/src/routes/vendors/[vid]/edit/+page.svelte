<script lang="ts">
  import VendorsEditor from "$lib/components/VendorsEditor.svelte";
  import type { VendorsPageData } from "$lib/svelte-types";
  let { data }: { data: VendorsPageData } = $props();
</script>

<VendorsEditor {data} />
