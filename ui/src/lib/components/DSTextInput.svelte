<script context="module">
  let idCounter = $state(0);
</script>

<script lang="ts" generics="T">
  import type { Snippet } from "svelte";

  // get an id for this instance from the counter in the module context then
  // increment it so the next instance gets a different id
  const thisId = idCounter;
  idCounter += 1;

  let {
    value = $bindable(),
    type = "text",
    step = undefined,
    min = undefined,
    max = undefined,
    errors,
    fieldName,
    uiName,
    placeholder,
    disabled = false,
  }: {
    value: string | number;
    type?: "text" | "number" | "password";
    step?: number;
    min?: number;
    max?: number;
    errors: Record<string, { message: string }>;
    fieldName: string;
    uiName: string;
    placeholder?: string;
    disabled?: boolean;
  } = $props();
</script>

<div class="flex w-full flex-col gap-2 {errors[fieldName] !== undefined ? 'bg-red-200' : ''}">
  <span class="flex w-full gap-2">
    <label for={`text-input-${thisId}`}>{uiName}</label>
    <input
      class="flex-1 rounded border border-neutral-300 px-1 {disabled ? 'opacity-50' : ''} {disabled
        ? 'cursor-not-allowed'
        : ''}"
      {type}
      step={step || null}
      min={min || null}
      max={max || null}
      id={`text-input-${thisId}`}
      name={fieldName}
      placeholder={placeholder || uiName}
      bind:value
      {disabled}
    />
  </span>
  {#if errors[fieldName] !== undefined}
    <span class="text-red-600">{errors[fieldName].message}</span>
  {/if}
</div>
