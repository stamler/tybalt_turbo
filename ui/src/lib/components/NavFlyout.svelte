<script lang="ts">
  import { resolve } from "$app/paths";
  import type { NavLink } from "$lib/navConfig";
  import { tick } from "svelte";

  let {
    label,
    items,
    pathname,
    claims,
    isBranchManager = false,
    showAllUi = false,
  }: {
    label: string;
    items: NavLink[];
    pathname: string;
    claims: string[];
    isBranchManager?: boolean;
    showAllUi?: boolean;
  } = $props();
  const id = $props.id();
  const visibleItems = $derived(
    items.filter((item) =>
      item.canView
        ? item.canView(claims, isBranchManager)
        : showAllUi || !item.requiredClaim || claims.includes(item.requiredClaim),
    ),
  );
  const isCurrent = (href: string) => pathname === href || pathname.startsWith(`${href}/`);
  const active = $derived(visibleItems.some((item) => isCurrent(item.href)));
  let trigger: HTMLButtonElement;
  let panel: HTMLElement;
  let open = $state(false);
  let position = $state({ top: 0, left: 0 });

  function placePanel() {
    const anchor = trigger.getBoundingClientRect();
    const bounds = panel.getBoundingClientRect();
    const fitsRight = anchor.right + bounds.width + 16 <= window.innerWidth;
    position = {
      left: fitsRight ? anchor.right + 8 : Math.max(8, window.innerWidth - bounds.width - 8),
      top: Math.max(
        8,
        Math.min(
          fitsRight ? anchor.top : anchor.bottom + 8,
          window.innerHeight - bounds.height - 8,
        ),
      ),
    };
  }

  async function onToggle(event: ToggleEvent) {
    open = event.newState === "open";
    if (open) {
      placePanel();
      await tick();
      if (!panel.matches(":popover-open")) return;
      (
        panel.querySelector<HTMLAnchorElement>('[aria-current="page"]') ??
        panel.querySelector<HTMLAnchorElement>("a")
      )?.focus();
    }
  }

  function onKeydown(event: KeyboardEvent) {
    const links = Array.from(panel.querySelectorAll<HTMLAnchorElement>("a"));
    const index = links.indexOf(document.activeElement as HTMLAnchorElement);
    let next: number;
    switch (event.key) {
      case "ArrowDown":
        next = (index + 1) % links.length;
        break;
      case "ArrowUp":
        next = (index - 1 + links.length) % links.length;
        break;
      case "Home":
        next = 0;
        break;
      case "End":
        next = links.length - 1;
        break;
      case "ArrowLeft":
        event.preventDefault();
        panel.hidePopover();
        trigger.focus();
        return;
      default:
        return;
    }
    event.preventDefault();
    links[next]?.focus();
  }

  $effect(() => {
    // Close the menu when navigation or access changes.
    void pathname;
    void visibleItems;
    panel?.hidePopover();
  });

  $effect(() => {
    if (!open) return;
    const dismiss = () => panel.hidePopover();
    const onScroll = (event: Event) => {
      if (!(event.target instanceof Node) || !panel.contains(event.target)) dismiss();
    };
    window.addEventListener("resize", dismiss);
    window.addEventListener("scroll", onScroll, true);
    return () => {
      window.removeEventListener("resize", dismiss);
      window.removeEventListener("scroll", onScroll, true);
    };
  });
</script>

<button
  bind:this={trigger}
  type="button"
  popovertarget={id}
  aria-expanded={open}
  aria-controls={id}
  class="ml-4 flex h-full min-w-0 grow items-center justify-between gap-2 rounded-sm px-2 text-xl hover:bg-neutral-600 focus-visible:outline-2 focus-visible:outline-white lg:text-sm"
  class:bg-neutral-600={active || open}
  onkeydown={(event) => {
    if (event.key === "ArrowRight" || event.key === "ArrowDown") {
      event.preventDefault();
      panel.showPopover();
    }
  }}
>
  <span>{label}</span>
  <svg
    aria-hidden="true"
    class="h-4 w-4"
    viewBox="0 0 20 20"
    fill="none"
    stroke="currentColor"
    stroke-width="1.5"
  >
    <path d="m8 5 5 5-5 5" stroke-linecap="round" stroke-linejoin="round" />
  </svg>
</button>

<!-- The top layer keeps the menu outside the sidebar's scroll boundary. -->
<nav
  bind:this={panel}
  {id}
  popover="auto"
  aria-label={`${label} views`}
  onbeforetoggle={(event) => {
    if (event.newState === "closed") open = false;
  }}
  ontoggle={onToggle}
  style:top={`${position.top}px`}
  style:left={`${position.left}px`}
  style:visibility={open ? "visible" : "hidden"}
  class="fixed inset-auto m-0 max-h-[calc(100dvh-1rem)] w-44 max-w-[calc(100vw-1rem)] overflow-y-auto rounded-md border border-neutral-500 bg-neutral-700 p-1 text-white shadow-lg"
>
  {#each visibleItems as item (item.href)}
    <a
      href={resolve(...([item.href] as Parameters<typeof resolve>))}
      onkeydown={onKeydown}
      onfocusout={(event) => {
        if (
          event.relatedTarget instanceof Node &&
          event.relatedTarget !== trigger &&
          !panel.contains(event.relatedTarget)
        )
          panel.hidePopover();
      }}
      aria-current={isCurrent(item.href) ? "page" : undefined}
      class="flex min-h-11 items-center rounded-sm px-3 text-base hover:bg-neutral-600 focus-visible:outline-2 focus-visible:outline-white lg:min-h-8 lg:text-sm"
      class:bg-neutral-600={isCurrent(item.href)}
      onclick={() => panel.hidePopover()}>{item.label}</a
    >
  {/each}
</nav>
