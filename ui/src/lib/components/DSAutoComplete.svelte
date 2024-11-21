<script context="module">
  let idCounter = $state(0);
</script>

<script lang="ts" generics="T">
  import type { Snippet } from "svelte";
  import MiniSearch from "minisearch";
  import type { SearchResult } from "minisearch";
  import DsActionButton from "./DSActionButton.svelte";

  // get an id for this instance from the counter in the module context then
  // increment it so the next instance gets a different id
  const thisId = idCounter;
  idCounter += 1;

  let {
    value = $bindable(),
    index,
    resultTemplate,
    errors,
    fieldName,
    uiName,
    disabled = false,
    idField = "id", // the field in the index that is the id, defaults to "id"
  }: {
    value: string;
    index: MiniSearch<T>;
    resultTemplate: Snippet<[SearchResult]>;
    errors: Record<string, { message: string }>;
    fieldName: string;
    uiName: string;
    disabled?: boolean;
    idField?: string;
  } = $props();

  let results = $state([] as SearchResult[]);
  let selectedIndex = $state(-1);

  function updateResults(event: Event) {
    const query = (event.target as HTMLInputElement).value;
    results = index.search(query, { prefix: true });
  }

  function keydown(event: KeyboardEvent) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      event.stopPropagation();
      // increment the selected index modulo the number of results
      selectedIndex = (selectedIndex + 1) % results.length;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      event.stopPropagation();
      // decrement the selected index modulo the number of results
      selectedIndex = (selectedIndex - 1 + results.length) % results.length;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      event.stopPropagation();
      if (selectedIndex !== -1) {
        value = results[selectedIndex][idField];
        results = [];
        selectedIndex = -1;
      }
    }
  }

  function getDocumentById(index: MiniSearch<T>, id: string) {
    const results = index.search(id.toString(), {
      fields: [idField],
      prefix: true,
      combineWith: "AND",
    });
    if (results.length === 0) {
      throw new Error(`No document found with ${idField} ${id}`);
    }
    return results[0];
  }

  function clearValue() {
    value = "";
  }

  const item = $derived.by(() => getDocumentById(index, value));
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="flex w-full flex-col gap-2" class:bg-red-200={errors[fieldName] !== undefined}>
  <div class="contents" onkeydown={keydown}>
    <span class="flex w-full gap-2">
      <label for={`autocomplete-input-${thisId}`}>{uiName}</label>
      {#if value !== ""}
        <span class={disabled ? "opacity-50" : ""}>{@render resultTemplate(item)}</span>
        {#if !disabled}
          <DsActionButton action={clearValue}>Clear</DsActionButton>
        {/if}
      {:else}
        <input
          class="flex-1 rounded border border-neutral-300 px-1 {disabled
            ? 'opacity-50'
            : ''} {disabled ? 'cursor-not-allowed' : ''}"
          type="text"
          id={`autocomplete-input-${thisId}`}
          name={fieldName}
          placeholder={uiName}
          oninput={updateResults}
          {disabled}
        />
      {/if}
    </span>

    {#if errors[fieldName] !== undefined}
      <span class="text-red-600">{errors[fieldName].message}</span>
    {/if}
    {#if results.length > 0}
      <ul class="suggestions">
        {#each results as choice, index}
          <li class="result" class:bg-blue-400={index === selectedIndex}>
            {@render resultTemplate(choice)}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>
