<script lang="ts">
  import type { Snippet } from "svelte";

  // One label and value in a DSFieldList: a small muted label stacked above
  // the value. An empty value hides the field unless showEmpty is set.
  let {
    label,
    value,
    href,
    hint,
    showEmpty = false,
    children,
  }: {
    label: string;
    value?: string | number | null;
    // Renders the value as a link.
    href?: string;
    // Muted suffix, e.g. "as of Oct 1, 2026".
    hint?: string;
    // Render "—" instead of hiding the field when the value is empty.
    showEmpty?: boolean;
    // A custom value (chips, buttons) instead of value.
    children?: Snippet;
  } = $props();

  const isEmpty = $derived(value === undefined || value === null || value === "");
</script>

{#if children || !isEmpty || showEmpty}
  <div class="min-w-0">
    <dt class="text-sm text-neutral-600">{label}</dt>
    <dd class="break-words">
      {#if children}
        {@render children()}
      {:else if isEmpty}
        —
      {:else if href}
        <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- callers pass any href -->
        <a class="link" {href}>{value}</a>
      {:else}
        {value}
      {/if}
      {#if hint}
        <span class="ml-1 text-sm text-neutral-600">{hint}</span>
      {/if}
    </dd>
  </div>
{/if}
