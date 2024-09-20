<script lang="ts">
  import { goto } from "$app/navigation";
  import Icon from "@iconify/svelte";
  import type { Snippet } from "svelte";

  let {
    icon,
    title,
    color,
    action,
    children,
    type = "button",
  }: {
    icon?: string;
    title?: string;
    color?: string;
    action?: (() => void) | string;
    children?: Snippet<[]>;
    type?: "button" | "submit" | "reset";
  } = $props();

  const normalizedAction = typeof action === "string" ? () => goto(action) : action;
  const normalizedColor = color ?? "yellow";
  const isIconContent = typeof icon === "string" && color !== undefined && title !== undefined;
  const isTextContent = children !== undefined;
</script>

{#if isIconContent}
  <span class="rounded-sm p-1 shadow-none hover:bg-yellow-100 active:shadow-inner">
    <button
      onclick={normalizedAction}
      {type}
      {title}
      class={`flex items-center text-neutral-500 hover:text-${color}-500 active:text-${color}-800`}
    >
      <Icon {icon} width="24px" />
    </button>
  </span>
{/if}
{#if isTextContent}
  <button
    {type}
    {title}
    onclick={normalizedAction}
    class="rounded-sm bg-{normalizedColor}-200 px-1 text-black hover:bg-{normalizedColor}-300 active:shadow-inner"
  >
    {@render children()}
  </button>
{/if}
