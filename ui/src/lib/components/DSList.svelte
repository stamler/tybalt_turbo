<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';

  let {
    items,
    search = false,
    listHeader = "",
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
    listHeader?: string;
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
    fields.push(item.id);
    return fields.join(",").toLowerCase();
  }

  const processedItems = $derived.by(() => {
    if (processorFn !== undefined && typeof processorFn === "function") {
      return processorFn(items.slice()); //.map((p) => toRaw(p)));
    }
    return items
      .slice() // shallow copy https://github.com/vuejs/vuefire/issues/244
      .filter((p) => searchString(p).indexOf(searchTerm.toLowerCase()) >= 0);
  });
</script>

<ul class="flex flex-col">
  {#if search && processorFn === undefined}
    <li id="listbar">
      <input id="searchbox" type="textbox" placeholder="search..." bind:value={searchTerm} />
      <span>{processedItems.length} items</span>
    </li>
  {/if}
  {#if listHeader !== ""}
    <li class="listheader">{listHeader}</li>
  {/if}
  {#each processedItems as item}
    <li class="flex even:bg-neutral-200 odd:bg-neutral-100">
      <div class="w-32">{@render anchor(item)}</div>
      <div class="flex flex-col w-full">
        <div class="headline_wrapper">
          <div class="headline">{@render headline(item)}</div>
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
        <div class="rowactionsbox">{@render actions(item)}</div>
      {/if}
    </li>
  {/each}
</ul>
