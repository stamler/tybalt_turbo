<script lang="ts">
  export interface TabItem {
    label: string;
    href: string;
    active?: boolean;
  }
  let { tabs }: { tabs: TabItem[] } = $props();
</script>

<div class="flex gap-4 rounded-t bg-neutral-100 px-2 py-1">
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
</div>
