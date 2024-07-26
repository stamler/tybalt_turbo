<script lang="ts" generics="T extends BaseSystemFields">
  import type { Snippet } from "svelte";
  import type { BaseSystemFields } from "$lib/pocketbase-types";
  import { groupBy } from "lodash";
  import DSInListHeader from "./DSInListHeader.svelte";
  let {
    items,
    search = false,
    inListHeader,
    groupHeader,
    groupField, // if groupField is set, group the items by this field
    groupFooter,
    processorFn,
    anchor,
    headline,
    byline,
    line1,
    line2,
    line3,
    actions,
  }: {
    items: T[];
    search?: boolean;
    inListHeader?: string;
    groupHeader?: Snippet<[string]>;
    groupField?: string;
    groupFooter?: Snippet<[string, T[]]>; // New group footer snippet that receives the group key and group items
    processorFn?: Function;
    anchor: Snippet<[T]>;
    headline: Snippet<[T]>;
    byline?: Snippet<[T]>;
    line1?: Snippet<[T]>;
    line2?: Snippet<[T]>;
    line3?: Snippet<[T]>;
    actions?: Snippet<[T]>;
  } = $props();

  let searchTerm = $state("");

  function searchString(item: T) {
    if (item === undefined || item === null) {
      return "";
    }
    const fields = Object.values(item);
    // if the item has an expand property, get all the keys from the expand
    // property, and then for each key, get Object.values(item.expand[key])
    // and add that to the fields array
    if (item.expand !== undefined) {
      const expandKeys = Object.keys(item.expand);
      expandKeys.forEach((key) => {
        if (item.expand !== undefined && item.expand[key] !== undefined) {
          fields.push(...Object.values(item.expand[key]));
        }
      });
    }
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
        class="flex-1 rounded border border-neutral-300 px-1"
      />
      {#if groupField === undefined}
        <span>{processedItems.length} items</span>
      {:else}
        <span>
          <!-- when grouping, get the sum of the length of the lists for every key in processed items -->
          {Object.keys(processedItems).reduce((acc, key) => acc + processedItems[key].length, 0)} items
        </span>
      {/if}
    </li>
  {/if}
  {#if inListHeader !== undefined}
    <DSInListHeader value={inListHeader} />
  {/if}

  {#snippet itemList(_processedItems)}
    {#each _processedItems as item}
      <li class="contents">
        <div class="col-span-3 grid grid-cols-subgrid bg-[inherit]">
          <div class="flex items-center justify-center px-4 py-2">{@render anchor(item)}</div>
          <div class="flex flex-col py-2">
            <div class="headline_wrapper">
              <div class="font-bold">{@render headline(item)}</div>
              {#if byline !== undefined}
                <div class="byline">{@render byline(item)}</div>
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

  {#if groupField !== undefined}
    {#each Object.keys(processedItems) as group}
      {#if groupHeader !== undefined}
        <DSInListHeader value={group} snippet={groupHeader} />
      {:else}
        <DSInListHeader value={group} />
      {/if}
      {@render itemList(processedItems[group])}
      {#if groupFooter !== undefined}
        <li class="contents">
          <div class="col-span-3 grid grid-cols-subgrid bg-[inherit]">
            {@render groupFooter(group, processedItems[group])}
          </div>
        </li>
      {/if}
    {/each}
  {:else}
    {@render itemList(processedItems)}
  {/if}
</ul>
