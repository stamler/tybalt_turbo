<script context="module">
  let idCounter = $state(0);
</script>

<script lang="ts" generics="T">
  // get an id for this instance from the counter in the module context then
  // increment it so the next instance gets a different id
  const thisId = idCounter;
  idCounter += 1;

  let {
    value = $bindable(),
    errors,
    fieldName,
    uiName,
    disabled = false,
  }: {
    value: boolean;
    errors: Record<string, { message: string }>;
    fieldName: string;
    uiName: string;
    disabled?: boolean;
  } = $props();
</script>

<div class="flex w-full flex-col gap-2 {errors[fieldName] !== undefined ? 'bg-red-200' : ''}">
  <span class="flex w-full gap-2">
    <input
      type="checkbox"
      class="h-4 w-4 {disabled ? 'opacity-50' : ''} {disabled ? 'cursor-not-allowed' : ''}"
      id={`checkbox-${thisId}`}
      name={fieldName}
      bind:checked={value}
      {disabled}
    />
    <label for={`checkbox-${thisId}`}>{uiName}</label>
  </span>
  {#if errors[fieldName] !== undefined}
    <span class="text-red-600">{errors[fieldName].message}</span>
  {/if}
</div>
