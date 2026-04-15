<script lang="ts">
  import DsLabel from "./DsLabel.svelte";

  let { count = 0 }: { count?: number } = $props();

  const tooltip = $derived(count === 1 ? "Shared with 1 user" : `Shared with ${count} users`);
</script>

{#if count > 0}
  <DsLabel color="cyan" title={tooltip}>Shared ({count})</DsLabel>
{/if}
