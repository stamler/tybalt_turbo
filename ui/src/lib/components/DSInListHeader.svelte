<script lang="ts">
  import type { Snippet } from "svelte";
  let { value, snippet }: { value: string; snippet?: Snippet<[string]> } = $props();
</script>

<li class="inlistheader col-span-3 bg-blue-300 p-1 text-neutral-700">
  {#if snippet !== undefined}
    {@render snippet(value)}
  {:else}
    {value}
  {/if}
</li>
