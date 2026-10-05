<script lang="ts">
  let {
    message,
    linkLabel,
    href,
    onShow,
    onExpire,
  }: {
    message: string;
    linkLabel: string;
    href?: string;
    onShow: () => void;
    onExpire: () => void;
  } = $props();
  let hovered = $state(false);
  let focused = $state(false);
  $effect(() => {
    if (hovered || focused) return;
    const timer = setTimeout(() => onExpire(), 6000);
    return () => clearTimeout(timer);
  });
</script>

<p
  role="status"
  class="flex flex-wrap items-center gap-2 text-sm text-green-800"
  onpointerenter={() => (hovered = true)}
  onpointerleave={() => (hovered = false)}
  onfocusin={() => (focused = true)}
  onfocusout={() => (focused = false)}
>
  <span>{message}</span>
  {#if href}<a
      {href}
      class="text-blue-700 underline"
      onclick={(event) => {
        event.preventDefault();
        onShow();
      }}>{linkLabel}</a
    >{/if}
</p>
