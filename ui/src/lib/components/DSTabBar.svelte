<script lang="ts">
  import type { Snippet } from "svelte";
  export interface TabItem {
    label: string;
    href: string;
    active?: boolean;
  }
  let { tabs, rightContent }: { tabs: TabItem[]; rightContent?: Snippet } = $props();
</script>

<div class="flex items-center gap-4 rounded-t bg-neutral-100 px-2 py-1">
  {#each tabs as t (t.href + t.label)}
    <!-- eslint-disable svelte/no-navigation-without-resolve -- callers pass any href -->
    <a
      href={t.href}
      class={`rounded-t px-3 py-1 ${
        t.active
          ? "bg-white font-semibold text-blue-700 shadow-sm"
          : "text-neutral-700 hover:bg-neutral-200"
      }`}
    >
      {t.label}
    </a>
    <!-- eslint-enable svelte/no-navigation-without-resolve -->
  {/each}
  {#if rightContent}
    <div class="ml-auto">
      {@render rightContent()}
    </div>
  {/if}
</div>
