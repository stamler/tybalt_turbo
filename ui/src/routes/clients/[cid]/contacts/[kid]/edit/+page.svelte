<script lang="ts">
  import ClientEditorPage from "$lib/components/clients/ClientEditorPage.svelte";
  let { data } = $props();
</script>

<ClientEditorPage {data} />
