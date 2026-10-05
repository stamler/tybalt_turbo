<script lang="ts">
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import { postalFields, type FieldErrors } from "$lib/clientEditors";
  import type { WorkflowRecord } from "$lib/clientEditors";
  let {
    item = $bindable(),
    errors,
    disabled = false,
  }: { item: WorkflowRecord; errors: FieldErrors; disabled?: boolean } = $props();
</script>

{#each Object.entries(postalFields) as [key, label]}
  <DsTextInput
    bind:value={item[key]}
    {errors}
    fieldName={key}
    uiName={`${label} (optional)`}
    {disabled}
  />
{/each}
