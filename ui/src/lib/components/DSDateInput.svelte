<script lang="ts">
  let {
    value = $bindable(""),
    id,
    name,
    class: className = "",
    min = undefined,
    max = undefined,
    disabled = false,
  }: {
    value: string;
    id?: string;
    name?: string;
    class?: string;
    min?: string;
    max?: string;
    disabled?: boolean;
  } = $props();
</script>

<input
  type="date"
  class={className}
  id={id ?? name}
  {name}
  min={min ?? undefined}
  max={max ?? undefined}
  bind:value
  {disabled}
/>
