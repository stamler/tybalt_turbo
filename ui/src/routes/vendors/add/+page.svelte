<script lang="ts">
  import type { VendorsPageData } from "$lib/svelte-types";
  import VendorsEditor from "$lib/components/VendorsEditor.svelte";
  let { data }: { data: VendorsPageData } = $props();
</script>

<VendorsEditor {data} />
