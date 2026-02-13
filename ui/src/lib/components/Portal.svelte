<script lang="ts">
  import { onMount } from "svelte";
  import type { Snippet } from "svelte";

  /*
   * Portal renders its children into a different DOM container (default: <body>)
   * while keeping Svelte component ownership unchanged.
   *
   * Why this exists:
   * - Avoid problematic DOM nesting (for example, modal/popover content inside forms).
   * - Escape parent clipping/stacking contexts created by overflow/z-index rules.
   * - Provide a lightweight primitive we can reuse for overlays in other editors.
   *
   * Where to reuse:
   * - Popovers, modals, dropdown panels, and other floating UI that should not be
   *   constrained by local layout containers.
   */
  let {
    target = "body",
    children,
  }: {
    target?: string;
    children: Snippet;
  } = $props();

  let host = $state<HTMLElement | null>(null);
  let portalNode = $state<HTMLDivElement | null>(null);

  onMount(() => {
    host = document.querySelector(target) ?? document.body;
    if (!host || !portalNode) return;
    host.appendChild(portalNode);

    return () => {
      portalNode?.remove();
    };
  });
</script>

<div bind:this={portalNode}>
  {@render children()}
</div>
