<script lang="ts">
  import Icon from "@iconify/svelte";

  let { text }: { text: string } = $props();
  let expanded = $state(false);
  const textId = $props.id();
  const instructions = $derived(text.trim());
  const long = $derived(instructions.length > 160 || instructions.split("\n").length > 3);
  const preview = $derived(
    instructions.split(/\s+/).slice(0, 12).join(" ").slice(0, 100).trimEnd(),
  );
  const action = $derived(
    expanded ? "Collapse invoice instructions" : "Expand invoice instructions",
  );
</script>

{#if instructions}
  <p class="text-sm">
    <span id={textId} class="whitespace-pre-wrap">{long && !expanded ? preview : instructions}</span
    >{#if long}<button
        type="button"
        class="ml-1 inline-flex min-h-6 min-w-6 cursor-pointer items-center justify-center rounded-sm align-middle text-blue-700 underline hover:bg-blue-100 focus-visible:outline-2 focus-visible:outline-blue-600"
        aria-label={action}
        title={action}
        aria-expanded={expanded}
        aria-controls={textId}
        onclick={() => (expanded = !expanded)}
      >
        {#if expanded}<Icon icon="mdi:chevron-up" width="18" aria-hidden="true" />{:else}<span
            aria-hidden="true">…</span
          >{/if}
      </button>{/if}
  </p>
{:else}
  <p class="text-sm text-neutral-600">No additional instructions</p>
{/if}
