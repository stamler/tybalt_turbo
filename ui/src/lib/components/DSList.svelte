<script lang="ts" generics="T extends { id: string; expand?: unknown }">
  // TODO: a better type parameter for BaseSystemFields than any?
  import type { Snippet } from "svelte";
  import { groupBy } from "lodash";
  import DSInListHeader from "./DSInListHeader.svelte";
  let {
    items,
    search = false,
    inListHeader,
    groupHeader,
    groupField, // if groupField is set, group the items by this field
    groupSort, // optional sorting order for group keys: 'ASC' | 'DESC'
    groupFooter,
    processorFn,
    anchor,
    headline,
    byline,
    line1,
    line2,
    line3,
    actions,
    searchBarExtra,
  }: {
    items: T[];
    search?: boolean;
    inListHeader?: string;
    groupHeader?: Snippet<[string]>;
    groupField?: string;
    groupSort?: "ASC" | "DESC";
    groupFooter?: Snippet<[string, T[]]>; // New group footer snippet that receives the group key and group items
    processorFn?: Function;
    anchor?: Snippet<[T]>;
    headline: Snippet<[T]>;
    byline?: Snippet<[T]>;
    line1?: Snippet<[T]>;
    line2?: Snippet<[T]>;
    line3?: Snippet<[T]>;
    actions?: Snippet<[T]>;
    searchBarExtra?: Snippet;
  } = $props();

  let searchTerm = $state("");

  function searchString(item: T) {
    if (item === undefined || item === null) {
      return "";
    }
    const fields = [] as string[];
    if (item.expand !== undefined) {
      // if the item has an expand property, get all the keys from the expand
      // property, and then for each key, get Object.values(item.expand[key])
      // and add that to the fields array
      const _ex = item.expand as Record<string, Record<string, any>>;
      if (_ex !== undefined) {
        const expandKeys = Object.keys(_ex);
        expandKeys.forEach((key) => {
          if (_ex[key] !== undefined && _ex[key] !== null) {
            const y = _ex[key];
            const vals = Object.values(y).filter((v) => v !== undefined && v !== null);
            fields.push(...vals);
          }
        });
      }
    }
    // get all the values from the item object and add them to the fields array
    const { expand, ...rest } = item;
    const vals = Object.values(rest)
      .filter((v) => v !== undefined && v !== null)
      .map((v) => v.toString());
    fields.push(...vals);

    return fields.join(",").toLowerCase();
  }

  const processedItems = $derived.by(() => {
    if (processorFn !== undefined && typeof processorFn === "function") {
      return processorFn(items.slice());
    }
    if (groupField !== undefined) {
      const filteredItems = items
        .slice()
        .filter((p) => searchString(p).indexOf(searchTerm.toLowerCase()) >= 0);
      return groupBy(filteredItems, groupField);
    }
    return items
      .slice() // shallow copy https://github.com/vuejs/vuefire/issues/244
      .filter((p) => searchString(p).indexOf(searchTerm.toLowerCase()) >= 0);
  });

  // groupKeys
  // Computes the ordered list of group headers to render when `groupField` is set.
  //
  // Behavior:
  // - If `groupSort` is undefined, preserve the natural key enumeration order
  //   from `Object.keys(processedItems)`, which corresponds to insertion order
  //   of the first occurrence of each group produced by lodash `groupBy`.
  // - If `groupSort` is provided ('ASC' | 'DESC'), sort group keys using a
  //   numeric-aware comparator:
  //     * If both keys can be parsed as finite numbers, compare numerically
  //       (e.g., "2" < "10").
  //     * Otherwise, fall back to case-insensitive `localeCompare` with
  //       `{ numeric: true }` so mixed strings like "A2" < "A10" sort as
  //       expected.
  //   The final order is reversed when `groupSort === 'DESC'`.
  //
  // Notes:
  // - Group keys originate from `groupBy(filteredItems, groupField)` and are
  //   strings; the comparator handles numeric-like strings gracefully.
  // - This only affects the order of group headers. The items within each
  //   group retain their original order from the input collection.
  const groupKeys = $derived.by(() => {
    if (groupField === undefined) return [] as string[];
    const keys = Object.keys(processedItems as Record<string, T[]>);
    if (groupSort === undefined) return keys;
    const cmp = (a: string, b: string) => {
      const an = Number(a);
      const bn = Number(b);
      if (!Number.isNaN(an) && !Number.isNaN(bn)) return an - bn;
      return a.localeCompare(b, undefined, { numeric: true, sensitivity: "base" });
    };
    const sorted = keys.slice().sort(cmp);
    return groupSort === "DESC" ? sorted.reverse() : sorted;
  });
</script>

<ul
  class="grid grid-cols-[auto_1fr_auto] [&>li:not(.inlistheader):nth-child(even)]:bg-neutral-100 [&>li:not(.inlistheader):nth-child(odd)]:bg-neutral-200"
>
  {#if search && processorFn === undefined}
    <li id="listbar" class="col-span-3 flex items-center gap-x-2 p-2">
      <input
        id="searchbox"
        type="textbox"
        placeholder="search..."
        bind:value={searchTerm}
        class="flex-1 rounded border border-neutral-300 px-1 py-1 text-base max-[639px]:px-2 max-[639px]:py-2 max-[639px]:text-lg"
      />
      {#if groupField === undefined}
        <span>{processedItems.length} items</span>
      {:else}
        <span>
          <!-- when grouping, get the sum of the length of the lists for every key in processed items -->
          {Object.keys(processedItems).reduce((acc, key) => acc + processedItems[key].length, 0)} items
        </span>
      {/if}
      {#if searchBarExtra}
        {@render searchBarExtra()}
      {/if}
    </li>
  {/if}
  {#if inListHeader !== undefined}
    <DSInListHeader value={inListHeader} />
  {/if}

  {#snippet itemList(_processedItems: T[])}
    {#each _processedItems as item}
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
            <div class="flex items-center gap-1 p-2">{@render actions(item)}</div>
          {/if}
        </div>
      </li>
    {/each}
  {/snippet}

  {#if groupField !== undefined}
    {#each groupKeys as group}
      {#if groupHeader !== undefined}
        <DSInListHeader value={group} snippet={groupHeader} />
      {:else}
        <DSInListHeader value={group} />
      {/if}
      {@render itemList(processedItems[group])}
      {#if groupFooter !== undefined}
        <li class="contents">
          <div class="col-span-3 grid grid-cols-subgrid items-center bg-[inherit]">
            {@render groupFooter(group, processedItems[group])}
          </div>
        </li>
      {/if}
    {/each}
  {:else}
    {@render itemList(processedItems)}
  {/if}
</ul>
