<script lang="ts">
  import type { Snippet } from "svelte";
  import DSCallout from "./DSCallout.svelte";

  let { message, children }: { message?: string; children?: Snippet } = $props();
</script>

<div class="m-2">
  <DSCallout tone="warning">
    {#if children}
      {@render children()}
    {:else if message}
      {message}
    {/if}
  </DSCallout>
</div>
