<script context="module">
  let idCounter = $state(0);
</script>

<script lang="ts" generics="T extends HasId">
  import type { Snippet } from "svelte";
  import type { HasId } from "$lib/pocketbase-types";
  import DsActionButton from "./DSActionButton.svelte";
  // get an id for this instance from the counter in the module context then
  // increment it so the next instance gets a different id
  const thisId = idCounter;
  idCounter += 1;

  let {
    value = $bindable(),
    items = [] as T[],
    errors,
    fieldName,
    uiName,
    clear = false,
    optionTemplate,
    disabled = false,
  }: {
    value: string | number;
    items: T[];
    errors: Record<string, { message: string }>;
    fieldName: string;
    uiName: string;
    clear?: boolean;
    optionTemplate: Snippet<[T]>;
    disabled?: boolean;
  } = $props();

  function clearValue() {
    value = "";
  }
</script>

<div class="flex w-full gap-2 {errors[fieldName] !== undefined ? 'bg-red-200' : ''}">
  <span class="flex w-full gap-2">
    <label for={`select-input-${thisId}`}>{uiName}</label>
    <select
      id={`select-input-${thisId}`}
      name={fieldName}
      bind:value
      class="rounded border border-neutral-300 px-1 {disabled ? 'opacity-50' : ''}"
      {disabled}
    >
      {#each items as item}
        <option value={item.id} selected={item.id === value}>{@render optionTemplate(item)}</option>
      {/each}
    </select>
    {#if clear === true && value !== undefined && value !== "" && !disabled}
      <DsActionButton action={clearValue} title="Clear">Clear</DsActionButton>
    {/if}
  </span>
  {#if errors[fieldName] !== undefined}
    <span class="text-red-600">{errors[fieldName].message}</span>
  {/if}
</div>
