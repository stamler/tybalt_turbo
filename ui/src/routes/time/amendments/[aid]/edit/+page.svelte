<script lang="ts">
  import type { PageData } from "./$types";
  import TimeAmendmentsEditor from "$lib/components/TimeAmendmentsEditor.svelte";
  let { data }: { data: PageData } = $props();
</script>

<TimeAmendmentsEditor {data} />
