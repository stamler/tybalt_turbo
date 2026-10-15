<script lang="ts">
  import type { Snippet } from "svelte";

  // Lays out DSCards: one column on phones, two from md, three from lg.
  let { children }: { children: Snippet } = $props();
</script>

<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
  {@render children()}
</div>
