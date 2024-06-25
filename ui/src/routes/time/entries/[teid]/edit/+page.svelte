<script lang="ts">
  import type { PageData } from "./$types";
  import TimeEntriesEditor from "$lib/components/TimeEntriesEditor.svelte";
  let { data }: { data: PageData } = $props();
</script>

<TimeEntriesEditor {data} />
