<script lang="ts">
  import { goto } from "$app/navigation";
  import Icon from "@iconify/svelte";
  import LoadingAnimation from "./LoadingAnimation.svelte";
  import type { Snippet } from "svelte";

  let {
    icon,
    title,
    color,
    transparentBackground = false,
    action,
    children,
    loading = false,
    type = "button",
    disabled = false,
  }: {
    icon?: string;
    title?: string;
    color?: string;
    transparentBackground?: boolean;
    action?: (() => void) | string;
    children?: Snippet<[]>;
    loading?: boolean;
    type?: "button" | "submit" | "reset";
    disabled?: boolean;
  } = $props();

  // Auto-loading: when the action returns a Promise, automatically disable the
  // button until it resolves/rejects. This prevents double-click race conditions
  // across the app (e.g. approving an expense twice) without requiring every
  // call site to manage its own loading state. The explicit `loading` prop from
  // the parent still works as an additional signal for external control.
  let actionInFlight = $state(false);

  function handleClick() {
    if (actionInFlight) return;
    // The action type is () => void, but callers may return a Promise in
    // practice (async functions). Cast to unknown so we can detect Promises.
    const result = normalizedAction?.() as unknown;
    // If the action returns a Promise, track it and auto-disable until settled.
    if (result instanceof Promise) {
      actionInFlight = true;
      result.finally(() => {
        actionInFlight = false;
      });
    }
  }

  const normalizedAction = $derived(
    typeof action === "string" ? () => goto(action) : action,
  );
  const isLoading = $derived(loading || actionInFlight);
  const normalizedColor = $derived(color ?? "yellow");
  const iconName = $derived(typeof icon === "string" ? icon : undefined);
  const isIconContent = $derived(
    iconName !== undefined && color !== undefined && title !== undefined,
  );
  const isTextContent = $derived(children !== undefined);
</script>

<button
  onclick={handleClick}
  {type}
  {title}
  disabled={isLoading || disabled}
  class="flex items-center rounded-xs {transparentBackground
    ? ''
    : 'bg-' + normalizedColor + '-200'} px-1 {isIconContent ? 'py-1' : 'py-0'} {isIconContent
    ? 'text-neutral-500'
    : 'text-black'} hover:bg-{normalizedColor}-300 hover:text-{normalizedColor}-500 active:text-{normalizedColor}-800 active:shadow-inner"
>
  {#if isLoading}
    <span class="flex h-6 w-5 items-center">
      <LoadingAnimation />
    </span>
  {:else if isIconContent}
    <Icon icon={iconName!} width="24px" class="flex h-6 items-center" />
  {:else if isTextContent}
    <span class="flex h-6 items-center">
      {@render children!()}
    </span>
  {/if}
</button>
