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

  let isOpen = $state(!collapsed);
  let isCollapsed = $derived(!isOpen);

  $effect(() => {
    isOpen = !collapsed;
  });
</script>

<details bind:open={isOpen} class="space-y-2">
  <summary class="cursor-pointer font-semibold text-neutral-700">{title}</summary>
  {#if !isCollapsed}
    {#if headerActions}
      <div class="flex min-h-9 items-center justify-end">
        {@render headerActions?.(isCollapsed)}
      </div>
    {/if}
    <div class="space-y-2">
      {@render children?.()}
    </div>
  {/if}
</details>
