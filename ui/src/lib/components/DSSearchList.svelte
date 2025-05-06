<script context="module">
  let idCounter = $state(0);
</script>

<script lang="ts" generics="T">
  import type { Snippet } from "svelte";
  import MiniSearch from "minisearch";
  import type { SearchResult } from "minisearch";

  // get an id for this instance from the counter in the module context then
  // increment it so the next instance gets a different id
  const thisId = idCounter;
  idCounter += 1;

  let results = $state([] as SearchResult[]);

  let {
    index,
    anchor,
    headline,
    byline,
    line1,
    line2,
    line3,
    actions,
    fieldName,
    uiName,
  }: {
    index: MiniSearch<T>;
    anchor?: Snippet<[T]>;
    headline: Snippet<[T]>;
    byline?: Snippet<[T]>;
    line1?: Snippet<[T]>;
    line2?: Snippet<[T]>;
    line3?: Snippet<[T]>;
    actions?: Snippet<[T]>;
    fieldName: string;
    uiName: string;
  } = $props();

  let searchTerm = $state("");

  function updateResults(event: Event) {
    const query = (event.target as HTMLInputElement).value;
    results = index.search(query, { prefix: true });
  }
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

  {#snippet itemList(_processedItems: T[])}
    {#each _processedItems as item}
      <li class="contents">
        <div class="col-span-3 grid grid-cols-subgrid bg-[inherit]">
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
  {/if}
</ul>
