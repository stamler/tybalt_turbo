<script lang="ts">
  import type { PageData } from "./$types";
  import JobsEditor from "$lib/components/JobsEditor.svelte";
  let { data }: { data: PageData } = $props();
</script>

{#key data.id}
  <JobsEditor {data} />
{/key}
