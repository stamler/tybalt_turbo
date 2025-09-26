<script lang="ts">
  import type { Snippet } from "svelte";
  import Icon from "@iconify/svelte";

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
  <div class="flex items-center justify-between gap-2">
    <button
      type="button"
      class="inline-flex min-h-9 items-center gap-2 text-left font-semibold"
      aria-expanded={!isCollapsed}
      onclick={toggle}
    >
      <span class="leading-none">{title}</span>
      <Icon
        icon="mdi:chevron-right"
        aria-hidden="true"
        class="h-5 w-5 text-neutral-500 transition-transform {isCollapsed ? '' : 'rotate-90'}"
      />
    </button>
    <div class="flex min-h-9 items-center">
      {@render headerActions?.(isCollapsed)}
    </div>
  </div>
  {#if !isCollapsed}
    <div class="pt-2">
      {@render children?.()}
    </div>
  {/if}
</section>
