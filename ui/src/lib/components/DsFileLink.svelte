<script lang="ts">
  import Icon from "@iconify/svelte";
  let {
    filename,
  }: {
    filename: string;
  } = $props();
</script>

<span class="flex gap-1 text-neutral-500 hover:text-neutral-800">
  {#if filename.toLowerCase().endsWith(".pdf")}
    <Icon icon="bxs:file-pdf" width="24px" />
  {:else if filename.toLowerCase().endsWith(".jpeg") || filename.toLowerCase().endsWith(".jpg")}
    <Icon icon="simple-icons:jpeg" width="24px" />
  {:else if filename.toLowerCase().endsWith(".png")}
    <Icon icon="bxs:file-png" width="24px" />
  {/if}
  <!-- {filename} -->
</span>
