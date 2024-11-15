<script lang="ts">
  import type { ClientsPageData } from "$lib/svelte-types";
  import ClientsEditor from "$lib/components/ClientsEditor.svelte";
  let { data }: { data: ClientsPageData } = $props();
</script>

<ClientsEditor {data} />
