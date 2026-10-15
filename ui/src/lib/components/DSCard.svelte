<script lang="ts">
  import type { Snippet } from "svelte";

  // A grey box with an optional title row. Place several in a DSCardGrid.
  let {
    title,
    actions,
    span = 1,
    children,
  }: {
    title?: string;
    // Right side of the title row.
    actions?: Snippet;
    // Column span inside a DSCardGrid.
    span?: 1 | 2 | "full";
    children: Snippet;
  } = $props();

  const spanClasses = {
    1: "",
    2: "md:col-span-2",
    full: "col-span-full",
  } as const;
</script>

<section class="min-w-0 space-y-3 rounded-sm bg-neutral-100 p-4 {spanClasses[span]}">
  {#if title || actions}
    <div class="flex flex-wrap items-center justify-between gap-2">
      {#if title}<h3 class="font-semibold">{title}</h3>{/if}
      {#if actions}
        <div class="ml-auto flex flex-wrap items-center gap-2">{@render actions()}</div>
      {/if}
    </div>
  {/if}
  {@render children()}
</section>
