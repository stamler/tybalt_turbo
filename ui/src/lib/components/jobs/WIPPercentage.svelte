<script lang="ts">
  let { percent }: { percent: number | null } = $props();
  const width = $derived(Math.max(0, Math.min(percent ?? 0, 100)));
  // Green to red through 100%; purple marks values above the project value.
  const hue = $derived(percent !== null && percent > 100 ? 275 : 120 * (1 - width / 100));
</script>

{#if percent === null}
  <span title="Some included amounts cannot be priced">—</span>
{:else}
  <div
    class="relative ml-auto h-6 w-28 overflow-hidden rounded-sm bg-neutral-100 ring-1 ring-neutral-200 ring-inset"
    title={`${percent.toFixed(1)}% of project value${percent > 100 ? " · exceeds project value" : ""}`}
  >
    <span
      aria-hidden="true"
      data-testid="wip-percentage-fill"
      class="absolute inset-y-0 left-0"
      style:width={`${width}%`}
      style:background={`linear-gradient(to right, hsl(${hue} 75% 90%), hsl(${hue} 70% 77%))`}
    ></span>
    <span
      class="relative flex h-full items-center justify-center font-semibold"
      style:color={`hsl(${hue} 75% 22%)`}>{percent.toFixed(1)}%</span
    >
  </div>
{/if}
