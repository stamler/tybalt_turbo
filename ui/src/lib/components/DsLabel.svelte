<script lang="ts">
  import type { Snippet } from "svelte";
  let { children, color } = $props<{
    children: Snippet<[string]>;
    color: string;
  }>();
</script>

<span
  class="flex items-center gap-1 rounded border border-{color}-300 bg-{color}-100 px-2 py-1 text-sm text-{color}-500"
>
  {@render children()}
</span>
