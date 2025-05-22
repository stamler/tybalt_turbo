<script lang="ts">
  import { goto } from "$app/navigation";
  import Icon from "@iconify/svelte";
  import LoadingAnimation from "./LoadingAnimation.svelte";
  import type { Snippet } from "svelte";

  let {
    icon,
    title,
    color,
    action,
    children,
    loading = false,
    type = "button",
  }: {
    icon?: string;
    title?: string;
    color?: string;
    action?: (() => void) | string;
    children?: Snippet<[]>;
    loading?: boolean;
    type?: "button" | "submit" | "reset";
  } = $props();

  const normalizedAction = typeof action === "string" ? () => goto(action) : action;
  const normalizedColor = color ?? "yellow";
  const isIconContent = typeof icon === "string" && color !== undefined && title !== undefined;
  const isTextContent = children !== undefined;
</script>

<button
  onclick={normalizedAction}
  {type}
  {title}
  disabled={loading}
  class="flex items-center rounded-sm bg-{normalizedColor}-200 px-1 {isIconContent
    ? 'py-1'
    : 'py-0'} {isIconContent
    ? 'text-neutral-500'
    : 'text-black'} hover:bg-{normalizedColor}-300 hover:text-{normalizedColor}-500 active:text-{normalizedColor}-800 active:shadow-inner"
>
  {#if loading}
    <span class="flex h-6 w-5 items-center">
      <LoadingAnimation />
    </span>
  {:else if isIconContent}
    <Icon {icon} width="24px" class="flex h-6 items-center" />
  {:else if isTextContent}
    <span class="flex h-6 items-center">
      {@render children()}
    </span>
  {/if}
</button>
