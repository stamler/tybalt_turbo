<script lang="ts">
  import Icon from "@iconify/svelte";
  import type { Snippet } from "svelte";

  // The one notice box. Pick the tone by meaning, not by colour.
  let {
    tone,
    title,
    icon,
    actions,
    children,
  }: {
    tone: "info" | "success" | "warning" | "danger";
    title?: string;
    // Defaults to an icon for the tone; false hides it.
    icon?: string | false;
    // Buttons, on the right or wrapped below.
    actions?: Snippet;
    children?: Snippet;
  } = $props();

  const tones = {
    info: { box: "border-blue-300 bg-blue-50 text-blue-900", icon: "mdi:information-outline" },
    success: {
      box: "border-green-300 bg-green-50 text-green-900",
      icon: "mdi:check-circle-outline",
    },
    warning: { box: "border-amber-300 bg-amber-50 text-amber-900", icon: "mdi:alert-outline" },
    danger: { box: "border-red-300 bg-red-50 text-red-900", icon: "mdi:alert-circle-outline" },
  } as const;

  const iconName = $derived(icon === false ? undefined : (icon ?? tones[tone].icon));
</script>

<div class="flex flex-wrap items-start gap-x-3 gap-y-2 rounded-sm border p-4 {tones[tone].box}">
  {#if iconName}
    <Icon icon={iconName} width="20px" class="mt-0.5 shrink-0" />
  {/if}
  <div class="min-w-0 flex-1 basis-48 space-y-1">
    {#if title}<p class="font-semibold">{title}</p>{/if}
    {@render children?.()}
  </div>
  {#if actions}
    <div class="ml-auto flex flex-wrap items-center gap-2">{@render actions()}</div>
  {/if}
</div>
