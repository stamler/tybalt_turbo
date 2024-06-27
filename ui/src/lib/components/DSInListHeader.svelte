<script lang="ts">
  import type { Snippet } from "svelte";
  let { value, snippet }: { value: string; snippet?: Snippet<[string]> } = $props();
</script>

<li class="inlistheader col-span-3 border-b border-blue-300 bg-blue-200 px-2 text-neutral-700">
  {#if snippet !== undefined}
    {@render snippet(value)}
  {:else}
    {value}
  {/if}
</li>
