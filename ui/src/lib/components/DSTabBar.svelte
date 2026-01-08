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
  {#each tabs as t}
    <a
      href={t.href}
      class={`rounded-t px-3 py-1 ${
        t.active
          ? "bg-white font-semibold text-blue-700 shadow"
          : "text-gray-700 hover:bg-neutral-200"
      }`}
    >
      {t.label}
    </a>
  {/each}
  {#if rightContent}
    <div class="ml-auto">
      {@render rightContent()}
    </div>
  {/if}
</div>
