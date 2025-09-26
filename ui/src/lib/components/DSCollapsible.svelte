<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title,
    collapsed = true,
    children,
    headerActions,
  }: {
    title: string;
    collapsed?: boolean;
    children?: Snippet<[]>;
    headerActions?: Snippet<[boolean]>;
  } = $props();

  let isCollapsed = $state(collapsed);

  function toggle() {
    isCollapsed = !isCollapsed;
  }
</script>

<section>
  <div class="flex items-center justify-between">
    <button
      type="button"
      class="inline-flex items-center gap-2 text-left font-semibold"
      aria-expanded={!isCollapsed}
      onclick={toggle}
    >
      <span>{title}</span>
      <span class="text-sm font-normal text-neutral-500">
        {isCollapsed ? "Show" : "Hide"}
      </span>
    </button>
    {@render headerActions?.(isCollapsed)}
  </div>
  {#if !isCollapsed}
    {@render children?.()}
  {/if}
</section>
