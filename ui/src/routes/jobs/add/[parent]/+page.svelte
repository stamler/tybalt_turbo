<script lang="ts">
  import JobsEditor from "$lib/components/JobsEditor.svelte";
  let { data } = $props();
</script>

<JobsEditor {data} />
