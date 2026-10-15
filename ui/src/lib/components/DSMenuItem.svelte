<script lang="ts">
  import Icon from "@iconify/svelte";
  import { onMount } from "svelte";
  import { getFlyoutContext } from "./dsFlyoutContext";

  // One row in a DSFlyout menu. A string action is a link; a function runs and
  // then closes the menu.
  let {
    label,
    icon,
    action,
    tone = "neutral",
  }: {
    label: string;
    icon?: string;
    action: (() => void) | string;
    tone?: "neutral" | "danger";
  } = $props();

  const flyout = getFlyoutContext();
  onMount(() => flyout?.register());

  const toneClasses = {
    neutral: "text-neutral-800",
    danger: "text-red-700",
  } as const;

  const classes = $derived(
    `flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-neutral-100 focus-visible:bg-neutral-100 focus-visible:outline-hidden ${toneClasses[tone]}`,
  );

  function run() {
    if (typeof action === "function") action();
    flyout?.close();
  }
</script>

{#if typeof action === "string"}
  <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- callers pass any href -->
  <a href={action} class={classes} onclick={() => flyout?.close()}>
    {#if icon}<Icon {icon} width="18px" />{/if}
    {label}
  </a>
{:else}
  <button type="button" class={classes} onclick={run}>
    {#if icon}<Icon {icon} width="18px" />{/if}
    {label}
  </button>
{/if}
