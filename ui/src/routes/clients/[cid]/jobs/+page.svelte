<script lang="ts">
  import type { PageData } from "./$types";
  import ClientWorkspacePage from "$lib/components/ClientWorkspacePage.svelte";
  let { data }: { data: PageData } = $props();
</script>

<ClientWorkspacePage {data} />
