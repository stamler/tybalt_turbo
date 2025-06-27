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
    excludeIds = [] as (string | number)[], // optional list of ids to exclude from results
    multi = false,
    choose,
  }: {
    value: string;
    index: MiniSearch<T>;
    resultTemplate: Snippet<[SearchResult]>;
    errors: Record<string, { message: string }>;
    fieldName: string;
    uiName: string;
    disabled?: boolean;
    idField?: string;
    excludeIds?: (string | number)[];
    multi?: boolean;
    choose?: (id: string | number) => void;
  } = $props();

  let results = $state([] as SearchResult[]);
  let selectedIndex = $state(-1);

  // reference to the internal input element so we can expose a focus helper
  // svelte-ignore non_reactive_update
  let inputElement: HTMLInputElement | null = null;
  // Allows parent components to focus the internal input element via `componentRef.focus()`.
  export function focus() {
    inputElement?.focus();
  }

  function updateResults(event: Event) {
    const query = (event.target as HTMLInputElement).value;
    // Search then filter out any excluded ids
    results = index
      .search(query, { prefix: true })
      .filter((r) => !excludeIds.includes(r[idField] as unknown as string | number));
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

    // Handle Enter key for both single and multi-select modes
    if (event.key === "Enter") {
      event.preventDefault();
      event.stopPropagation();
      if (selectedIndex !== -1) {
        const chosen = results[selectedIndex][idField] as unknown as string | number;

        if (multi) {
          // Call the provided callback if available
          choose?.(chosen);

          // Remove chosen item from current results so list stays open.
          results = results.filter((r) => r[idField] !== chosen);
          selectedIndex = -1;
        } else {
          value = chosen as unknown as string;
          results = [];
          selectedIndex = -1;
        }
      } else if (multi) {
        // In multi-select mode, Enter without selection closes the list
        results = [];
        selectedIndex = -1;
        // Clear the current input value so user can start a fresh search
        if (inputElement) inputElement.value = "";
      }
    }

    // Handle space key for multi-select mode (only when item is selected)
    if (multi && event.key === " " && selectedIndex !== -1) {
      event.preventDefault();
      event.stopPropagation();
      const chosen = results[selectedIndex][idField] as unknown as string | number;

      // Call the provided callback if available
      choose?.(chosen);

      // Remove chosen item from current results so list stays open.
      results = results.filter((r) => r[idField] !== chosen);
      selectedIndex = -1;
    }

    // Escape always closes the list in either mode
    if (event.key === "Escape") {
      results = [];
      selectedIndex = -1;
    }
  }

  function getDocumentById(index: MiniSearch<T>, id: string) {
    // Ensure we don't return a document that has been excluded
    const results = index.search(id.toString(), {
      fields: [idField],
      prefix: true,
      combineWith: "AND",
    });
    const filtered = results.filter(
      (r) => !excludeIds.includes(r[idField] as unknown as string | number),
    );
    if (filtered.length === 0) {
      // Return a minimal stub so the component can render without error.
      const stub: any = { [idField]: id };
      return stub as unknown as SearchResult;
    }
    return filtered[0];
  }

  function clearValue() {
    value = "";
  }

  const item = $derived.by(() => getDocumentById(index, value));

  function commitChoice(chosen: string | number) {
    if (multi) {
      choose?.(chosen);
      results = results.filter((r) => r[idField] !== chosen);
      selectedIndex = -1;
    } else {
      value = chosen as unknown as string;
      results = [];
      selectedIndex = -1;
    }
    // ensure keyboard interaction continues after mouse selection
    inputElement?.focus();
  }
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
          bind:this={inputElement}
        />
      {/if}
    </span>

    {#if errors[fieldName] !== undefined}
      <span class="text-red-600">{errors[fieldName].message}</span>
    {/if}
    {#if results.length > 0}
      <ul class="suggestions">
        {#each results as choice, index}
          <li
            class="result cursor-pointer"
            class:bg-blue-400={index === selectedIndex}
            onmouseenter={() => (selectedIndex = index)}
          >
            <div
              role="button"
              tabindex="0"
              class="w-full"
              onclick={() => commitChoice(choice[idField])}
              onkeydown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  commitChoice(choice[idField]);
                }
              }}
            >
              {@render resultTemplate(choice)}
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>
