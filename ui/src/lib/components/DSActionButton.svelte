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
    variant = "chip",
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
    // chip: the compact coloured button used across the app.
    // primary: filled blue, for form submits and dialog confirms.
    // danger: filled red, for destructive confirms.
    // secondary: filled grey, for Cancel beside a primary or danger button.
    // plain: text only, styled like a link.
    variant?: "chip" | "primary" | "danger" | "secondary" | "plain";
  } = $props();

  // Full class strings so Tailwind sees every class. `bg` is dropped when
  // transparentBackground is set.
  const chipColors: Record<string, { bg: string; interact: string }> = {
    yellow: {
      bg: "bg-yellow-200",
      interact: "hover:bg-yellow-300 hover:text-yellow-500 active:text-yellow-800",
    },
    red: { bg: "bg-red-200", interact: "hover:bg-red-300 hover:text-red-500 active:text-red-800" },
    green: {
      bg: "bg-green-200",
      interact: "hover:bg-green-300 hover:text-green-500 active:text-green-800",
    },
    blue: {
      bg: "bg-blue-200",
      interact: "hover:bg-blue-300 hover:text-blue-500 active:text-blue-800",
    },
    orange: {
      bg: "bg-orange-200",
      interact: "hover:bg-orange-300 hover:text-orange-500 active:text-orange-800",
    },
    purple: {
      bg: "bg-purple-200",
      interact: "hover:bg-purple-300 hover:text-purple-500 active:text-purple-800",
    },
    cyan: {
      bg: "bg-cyan-200",
      interact: "hover:bg-cyan-300 hover:text-cyan-500 active:text-cyan-800",
    },
    teal: {
      bg: "bg-teal-200",
      interact: "hover:bg-teal-300 hover:text-teal-500 active:text-teal-800",
    },
    neutral: {
      bg: "bg-neutral-200",
      interact: "hover:bg-neutral-300 hover:text-neutral-500 active:text-neutral-800",
    },
    gray: {
      bg: "bg-gray-200",
      interact: "hover:bg-gray-300 hover:text-gray-500 active:text-gray-800",
    },
  };

  const filledBase =
    "flex items-center justify-center gap-1 rounded-sm px-4 py-2 disabled:opacity-50";
  const variantClasses = {
    primary: `${filledBase} bg-blue-500 text-white hover:bg-blue-600`,
    danger: `${filledBase} bg-red-700 text-white hover:bg-red-800`,
    secondary: `${filledBase} bg-neutral-200 text-neutral-700 hover:bg-neutral-300`,
    plain:
      "inline-flex items-center gap-1 text-blue-700 hover:underline disabled:no-underline disabled:opacity-50",
  } as const;

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

  // eslint-disable-next-line svelte/no-navigation-without-resolve -- callers pass any path
  const normalizedAction = $derived(typeof action === "string" ? () => goto(action) : action);
  const isLoading = $derived(loading || actionInFlight);
  const chipColor = $derived(chipColors[color ?? "yellow"] ?? chipColors.yellow);
  const iconName = $derived(typeof icon === "string" ? icon : undefined);
  const isIconContent = $derived(
    iconName !== undefined && color !== undefined && title !== undefined,
  );
  const isTextContent = $derived(children !== undefined);
  const chipClasses = $derived(
    `flex items-center rounded-xs ${transparentBackground ? "" : chipColor.bg} px-1 ${
      isIconContent ? "py-1 text-neutral-500" : "py-0 text-black"
    } ${chipColor.interact} active:shadow-inner`,
  );
</script>

{#if variant === "chip"}
  <button onclick={handleClick} {type} {title} disabled={isLoading || disabled} class={chipClasses}>
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
{:else}
  <button
    onclick={handleClick}
    {type}
    {title}
    disabled={isLoading || disabled}
    class={variantClasses[variant]}
  >
    {#if isLoading}
      <span class="flex h-5 w-5 items-center">
        <LoadingAnimation />
      </span>
    {:else if iconName}
      <Icon icon={iconName} width={variant === "plain" ? "18px" : "24px"} />
    {/if}
    {@render children?.()}
  </button>
{/if}
