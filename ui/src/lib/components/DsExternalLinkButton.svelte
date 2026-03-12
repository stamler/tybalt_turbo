<script lang="ts">
  /* eslint-disable svelte/no-navigation-without-resolve */
  import Icon from "@iconify/svelte";

  let {
    href,
    label,
    icon = "mdi:download",
  }: {
    href: string;
    label: string;
    icon?: string;
  } = $props();
</script>

<a
  {href}
  target="_blank"
  rel="noopener noreferrer"
  class="inline-flex items-center gap-1 rounded-sm bg-blue-600 px-2 py-1 text-sm text-white hover:bg-blue-700"
>
  <Icon {icon} width="16" />
  {label}
</a>
