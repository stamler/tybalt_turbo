<script lang="ts">
  import Icon from "@iconify/svelte";

  let {
    icon,
    title,
    color,
    action,
  }: {
    icon: string;
    title: string;
    color: string;
    action: (() => void) | string;
  } = $props();

  const isLink = typeof action === "string";
</script>

<span class="rounded-sm p-1 shadow-none hover:bg-yellow-100 active:shadow-inner">
  {#if isLink}
    <a
      href={action}
      {title}
      class={`flex items-center text-neutral-500 hover:text-${color}-500 active:text-${color}-800`}
    >
      <Icon {icon} width="24px" />
    </a>
  {:else}
    <button
      onclick={action}
      type="button"
      {title}
      class={`flex items-center text-neutral-500 hover:text-${color}-500 active:text-${color}-800`}
    >
      <Icon {icon} width="24px" />
    </button>
  {/if}
</span>
