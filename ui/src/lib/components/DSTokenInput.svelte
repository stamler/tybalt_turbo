<script module>
  /* 
  This is a token input UI element. It is a text input that is used to input a
  list of strings. The input accepts regular text input and when a comma is
  entered, the current text is added to the list of strings and the input is
  cleared. The list of strings is stored in the `value` prop. The existing list
  of strings is displayed as a series of pill-like elements. Each pill can be
  removed by clicking the 'x' icon on the right side of the pill. Backspace can
  also be used to remove the last pill. 
  */
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
  }: {
    value: string[];
    errors: Record<string, { message: string }>;
    fieldName: string;
    uiName: string;
  } = $props();

  let inputValue = $state("");

  function removeToken(index: number) {
    value = value.filter((_, i) => i !== index);
  }

  function addToken(val: string) {
    if (val.trim() === "") {
      return;
    }
    value = [...value, val.trim()];
    inputValue = "";
  }

  function preventDefault(fn: (event: Event) => void) {
    return (event: Event) => {
      event.preventDefault();
      fn(event);
    };
  }

  function handleInput(event: Event) {
    const inputElement = event.target as HTMLInputElement;
    const value = inputElement.value || "";
    if (value.endsWith(",")) {
      addToken(value.slice(0, -1));
    } else {
      inputValue = value;
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Backspace" && inputValue === "" && value.length > 0) {
      removeToken(value.length - 1);
      event.preventDefault();
    }
  }
</script>

<div class="flex w-full flex-col gap-2 {errors[fieldName] !== undefined ? 'bg-red-200' : ''}">
  <span class="flex w-full gap-2">
    <label class="flex shrink-0 items-center" for={`text-input-${thisId}`}>{uiName}</label>
    <div
      class="focus-within:border-blue-500, flex w-full flex-wrap gap-1 rounded-sm border border-neutral-300 bg-white p-1 focus-within:ring-2 focus-within:ring-blue-500"
    >
      {#each value as item, i}
        <span class="flex items-center rounded-full bg-neutral-200 px-2">
          <span>{item}</span>
          <button class="text-neutral-500" onclick={preventDefault(() => removeToken(i))}>
            &times;
          </button>
        </span>
      {/each}
      <input
        class="flex-1 focus:outline-hidden"
        id={`text-input-${thisId}`}
        type="text"
        name={fieldName}
        placeholder={uiName}
        onkeydown={handleKeydown}
        oninput={handleInput}
        bind:value={inputValue}
      />
    </div>
  </span>
  {#if errors[fieldName] !== undefined}
    <span class="text-red-600">{errors[fieldName].message}</span>
  {/if}
</div>
