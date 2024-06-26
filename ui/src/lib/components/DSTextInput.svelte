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
  }: {
    value: string | number;
    type?: "text" | "number" | "password";
    step?: number;
    min?: number;
    max?: number;
    errors: Record<string, { message: string }>;
    fieldName: string;
    uiName: string;
  } = $props();
</script>

<div class="flex flex-col w-full gap-2 {errors[fieldName] !== undefined ? 'bg-red-200' : ''}">
  <span class="flex w-full gap-2">
    <label for={`text-input-${thisId}`}>{uiName}</label>
    <input
      class="flex-1 border rounded border-neutral-300 px-1"
      {type}
      step={step || null}
      min={min || null}
      max={max || null}
      id={`text-input-${thisId}`}
      name={fieldName}
      placeholder={uiName}
      bind:value
    />
  </span>
  {#if errors[fieldName] !== undefined}
    <span class="text-red-600">{errors[fieldName].message}</span>
  {/if}
</div>
