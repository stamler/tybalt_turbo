<script lang="ts">
  let {
    value = $bindable(),
    options,
  }: {
    value: string;
    options: { id: string; label: string }[];
  } = $props();
</script>

<div class="flex overflow-hidden rounded border border-neutral-300">
  {#each options as option}
    <button
      type="button"
      onclick={() => (value = option.id)}
      class="px-3 py-1 text-sm {value === option.id
        ? 'bg-blue-500 text-white'
        : 'bg-white text-gray-700 hover:bg-neutral-100'}"
    >
      {option.label}
    </button>
  {/each}
</div>
