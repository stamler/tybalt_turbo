<script lang="ts">
  import Icon from "@iconify/svelte";
  import type { Snippet } from "svelte";

  let {
    label,
    title,
    children,
    warning = false,
  }: {
    label?: string;
    title: string;
    children: Snippet;
    warning?: boolean;
  } = $props();
  const id = $props.id();
  let trigger: HTMLButtonElement;
  let panel: HTMLDivElement;
  let open = $state(false);
  let position = $state({ top: 0, left: 0 });

  function placePanel() {
    const anchor = trigger.getBoundingClientRect();
    const bounds = panel.getBoundingClientRect();
    const below = anchor.bottom + 8;
    const above = anchor.top - bounds.height - 8;
    position = {
      top: Math.max(
        12,
        Math.min(
          below + bounds.height <= window.innerHeight ? below : above,
          window.innerHeight - bounds.height - 12,
        ),
      ),
      left: Math.max(
        12,
        Math.min(anchor.right - bounds.width, window.innerWidth - bounds.width - 12),
      ),
    };
  }

  function onToggle(event: ToggleEvent) {
    open = event.newState === "open";
    if (open) placePanel();
  }

  $effect(() => {
    if (!open) return;
    const dismissOnScroll = (event: Event) => {
      if (!(event.target instanceof Node) || !panel.contains(event.target)) panel.hidePopover();
    };
    window.addEventListener("resize", placePanel);
    window.addEventListener("scroll", dismissOnScroll, true);
    return () => {
      window.removeEventListener("resize", placePanel);
      window.removeEventListener("scroll", dismissOnScroll, true);
    };
  });
</script>

<button
  bind:this={trigger}
  type="button"
  popovertarget={id}
  aria-label={label ? `${label}: ${title}` : title}
  aria-haspopup="dialog"
  aria-controls={id}
  aria-expanded={open}
  class="inline-flex min-h-8 min-w-8 items-center justify-center gap-1 rounded-sm px-1.5 py-1 text-sm hover:bg-neutral-100 focus-visible:outline-2 focus-visible:outline-blue-600 {warning
    ? 'text-amber-800'
    : 'text-neutral-600'}"
>
  {#if label}<span>{label}</span>{/if}
  <Icon icon="mdi:help-circle-outline" width="16" aria-hidden="true" />
</button>

<!-- Native popovers use the top layer, so table overflow cannot clip them.
     The browser handles outside clicks, Escape, and keyboard focus return. -->
<div
  bind:this={panel}
  {id}
  popover="auto"
  role="dialog"
  aria-label={title}
  ontoggle={onToggle}
  style:top={`${position.top}px`}
  style:left={`${position.left}px`}
  style:visibility={open ? "visible" : "hidden"}
  class="fixed inset-auto m-0 max-h-[calc(100dvh-1.5rem)] w-80 max-w-[calc(100vw-1.5rem)] overflow-y-auto rounded-md border border-neutral-300 bg-white p-4 text-left text-sm font-normal text-neutral-800 shadow-lg"
>
  <div class="mb-2 flex items-start justify-between gap-3">
    <span class="font-semibold">{title}</span>
    <button
      type="button"
      popovertarget={id}
      popovertargetaction="hide"
      aria-label="Close explanation"
      class="inline-flex min-h-8 min-w-8 items-center justify-center rounded-sm hover:bg-neutral-100"
    >
      <Icon icon="mdi:close" width="18" aria-hidden="true" />
    </button>
  </div>
  <div class="space-y-2">{@render children()}</div>
</div>
