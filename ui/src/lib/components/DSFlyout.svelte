<script lang="ts">
  import type { Snippet } from "svelte";
  import { setFlyoutContext } from "./dsFlyoutContext";

  // An anchored panel that closes on an outside click or Escape. The trigger
  // snippet renders the control that opens it, usually a DSActionButton.
  let {
    open = $bindable(false),
    label,
    align = "start",
    // Hide the trigger while no DSMenuItem is inside (used by DSPageHeader's
    // menu, which may render nothing for people without the right claims).
    hideWhenEmpty = false,
    trigger,
    children,
  }: {
    open?: boolean;
    label: string;
    align?: "start" | "end";
    hideWhenEmpty?: boolean;
    trigger: Snippet<[{ toggle: () => void; open: boolean }]>;
    children: Snippet;
  } = $props();

  let root: HTMLElement | undefined = $state();
  let itemCount = $state(0);

  function toggle() {
    open = !open;
  }

  setFlyoutContext({
    close: () => (open = false),
    register: () => {
      itemCount++;
      return () => itemCount--;
    },
  });

  function handlePointerDown(event: PointerEvent) {
    if (open && root && !root.contains(event.target as Node)) open = false;
  }

  function handleKeydown(event: KeyboardEvent) {
    if (open && event.key === "Escape") open = false;
  }
</script>

<svelte:window onpointerdown={handlePointerDown} onkeydown={handleKeydown} />

<div bind:this={root} class={hideWhenEmpty && itemCount === 0 ? "hidden" : "relative inline-block"}>
  {@render trigger({ toggle, open })}
  <!-- Kept mounted while closed so menu items can register. -->
  <div
    role="group"
    aria-label={label}
    hidden={!open}
    class="absolute top-full z-30 mt-1 max-w-[calc(100vw-2rem)] min-w-48 rounded-sm border border-neutral-300 bg-white py-1 text-left shadow-lg {align ===
    'end'
      ? 'right-0'
      : 'left-0'}"
  >
    {@render children()}
  </div>
</div>
