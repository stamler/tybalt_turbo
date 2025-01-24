<script lang="ts">
  import type { Snippet } from "svelte";
  let { children, color, title, style } = $props<{
    children: Snippet<[string]>;
    color: string;
    title?: string;
    style?: string;
  }>();
</script>

<!-- if the style is "inverted", show the inverted style. Otherwise show the regular style -->
{#if style === "inverted"}
  <span
    class="flex items-center gap-1 rounded-full bg-{color}-500 px-2 py-1 text-sm text-white"
    {title}
  >
    {@render children()}
  </span>
{:else}
  <span
    class="flex items-center gap-1 rounded border border-{color}-300 bg-{color}-100 px-2 py-1 text-sm text-{color}-500"
    {title}
  >
    {@render children()}
  </span>
{/if}
