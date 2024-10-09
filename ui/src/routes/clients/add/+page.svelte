<script lang="ts">
  import type { PageData } from "./$types";
  import ClientsEditor from "$lib/components/ClientsEditor.svelte";
  let { data }: { data: PageData } = $props();
</script>

<ClientsEditor {data} />
