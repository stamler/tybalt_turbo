<script lang="ts">
  import type { Snippet } from "svelte";

  // Wraps DSFields in a <dl>. columns={2} uses two columns from md up.
  let { columns = 1, children }: { columns?: 1 | 2; children: Snippet } = $props();

  const columnClasses = {
    1: "",
    2: "md:grid-cols-2",
  } as const;
</script>

<dl class="grid gap-x-6 gap-y-3 {columnClasses[columns]}">
  {@render children()}
</dl>
