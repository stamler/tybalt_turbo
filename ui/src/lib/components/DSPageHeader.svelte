<script lang="ts">
  import type { Snippet } from "svelte";
  import DSActionButton from "./DSActionButton.svelte";
  import DSFlyout from "./DSFlyout.svelte";

  // The top of a detail page. The title is the record itself ("25-123",
  // "Jane Doe"), not "X Details".
  let {
    title,
    subtitle,
    badges,
    context,
    actions,
    menu,
  }: {
    title: string;
    subtitle?: string;
    // Status and kind labels beside the title.
    badges?: Snippet;
    // One muted line, e.g. "Subjob of 25-100 · from proposal P25-033".
    context?: Snippet;
    // Frequent actions, right-aligned DSActionButtons.
    actions?: Snippet;
    // Rare and admin actions as DSMenuItems in a "⋯" flyout. The button only
    // shows when at least one item renders.
    menu?: Snippet;
  } = $props();
</script>

<header class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
  <div class="min-w-0 space-y-1">
    <div class="flex flex-wrap items-center gap-2">
      <h1 class="text-2xl font-bold break-words">{title}</h1>
      {#if badges}{@render badges()}{/if}
    </div>
    {#if subtitle}
      <p class="text-lg text-neutral-700">{subtitle}</p>
    {/if}
    {#if context}
      <p class="text-sm text-neutral-600">{@render context()}</p>
    {/if}
  </div>
  {#if actions || menu}
    <div class="flex shrink-0 flex-wrap items-center gap-2">
      {@render actions?.()}
      {#if menu}
        <DSFlyout label="More actions" align="end" hideWhenEmpty>
          {#snippet trigger({ toggle })}
            <DSActionButton
              icon="mdi:dots-horizontal"
              title="More actions"
              color="neutral"
              action={toggle}
            />
          {/snippet}
          {@render menu()}
        </DSFlyout>
      {/if}
    </div>
  {/if}
</header>
