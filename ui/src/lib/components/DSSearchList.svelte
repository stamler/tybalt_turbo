<script context="module">
  let idCounter = $state(0);
</script>

<script lang="ts" generics="T">
  import type { Snippet } from "svelte";
  import MiniSearch from "minisearch";
  import type { SearchResult } from "minisearch";
  import DSInListHeader from "./DSInListHeader.svelte";
  import { collectionEvents } from "$lib/stores/collectionEvents";
  import { onMount } from "svelte";

  const MAX_RESULTS = 100; // don't render more than 100 results
  const MIN_SEARCH_LENGTH = 2; // don't search if term is less than 2 characters

  // get an id for this instance from the counter in the module context then
  // increment it so the next instance gets a different id
  const thisId = idCounter;
  idCounter += 1;

  let results = $state([] as SearchResult[]);

  let {
    index,
    inListHeader,
    anchor,
    headline,
    byline,
    line1,
    line2,
    line3,
    actions,
    fieldName,
    uiName,
    collectionName,
  }: {
    index: MiniSearch<T>;
    inListHeader?: string;
    anchor?: Snippet<[T]>;
    headline: Snippet<[T]>;
    byline?: Snippet<[T]>;
    line1?: Snippet<[T]>;
    line2?: Snippet<[T]>;
    line3?: Snippet<[T]>;
    actions?: Snippet<[T]>;
    fieldName: string;
    uiName: string;
    collectionName?: string;
  } = $props();

  let searchTerm = $state("");

  function updateResults(event?: Event) {
    // if an event is provided, update the search term from the input
    if (event) {
      searchTerm = (event.target as HTMLInputElement).value;
    }

    // if the search term is less than 3 characters, don't search and reset the
    // results
    if (searchTerm.length < MIN_SEARCH_LENGTH) {
      results = [];
      return;
    }
    results = index.search(searchTerm, { prefix: true });
  }

  // Subscribe to collection events and refresh when relevant events occur
  onMount(() => {
    const unsubscribe = collectionEvents.subscribe((event) => {
      if (event && event.collection === collectionName && searchTerm.length >= MIN_SEARCH_LENGTH) {
        // Only refresh if we have an active search
        updateResults();
      }
    });

    return unsubscribe;
  });
</script>

<ul
  class="grid grid-cols-[auto_1fr_auto] [&>li:not(.inlistheader):nth-child(even)]:bg-neutral-100 [&>li:not(.inlistheader):nth-child(odd)]:bg-neutral-200"
>
  <li id="listbar" class="col-span-3 flex items-center gap-x-2 p-2">
    <input
      class="flex-1 rounded border border-neutral-300 px-1"
      type="text"
      id={`autocomplete-input-${thisId}`}
      name={fieldName}
      placeholder={uiName}
      oninput={updateResults}
    />
    <span>{results.length} items</span>
  </li>
  {#if inListHeader !== undefined}
    <DSInListHeader value={inListHeader} />
  {/if}
  {#snippet itemList(_items: T[])}
    {#each _items.slice(0, MAX_RESULTS) as item}
      <li class="contents">
        <div class="col-span-3 grid grid-cols-subgrid items-center bg-[inherit]">
          {#if anchor !== undefined}
            <div class="flex min-w-24 items-center justify-center p-2">
              {@render anchor(item)}
            </div>
          {:else}
            <div class="w-4"></div>
          {/if}
          <div class="flex flex-col py-2">
            <div class="headline_wrapper flex items-center gap-2">
              <span class="font-bold">{@render headline(item)}</span>
              {#if byline !== undefined}
                <span class="byline">{@render byline(item)}</span>
              {/if}
            </div>
            {#if line1 !== undefined}
              <div class="firstline">{@render line1(item)}</div>
            {/if}
            {#if line2 !== undefined}
              <div class="secondline">{@render line2(item)}</div>
            {/if}
            {#if line3 !== undefined}
              <div class="thirdline">{@render line3(item)}</div>
            {/if}
          </div>
          {#if actions !== undefined}
            <div class="flex items-center gap-1 px-2 py-2">{@render actions(item)}</div>
          {/if}
        </div>
      </li>
    {/each}
  {/snippet}

  {#if results.length > 0}
    {@render itemList(results.map((searchResult) => searchResult as unknown as T))}
  {:else if searchTerm.length >= MIN_SEARCH_LENGTH}
    <li class="col-span-3">
      <div class="flex items-center justify-center p-2">
        <span class="text-lg text-neutral-500">No results found</span>
      </div>
    </li>
  {:else}
    <li class="col-span-3">
      <div class="flex items-center justify-center p-2">
        <span class="text-lg text-neutral-500">Search for something</span>
      </div>
    </li>
  {/if}
</ul>
