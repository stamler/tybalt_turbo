<script lang="ts">
  import type { Snippet } from "svelte";

  let { message, children }: { message?: string; children?: Snippet } = $props();
</script>

<div class="m-2 rounded-sm border border-yellow-500 bg-yellow-50 p-4 text-yellow-800">
  {#if children}
    {@render children()}
  {:else if message}
    {message}
  {/if}
</div>
