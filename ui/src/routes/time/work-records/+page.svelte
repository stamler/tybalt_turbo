<script lang="ts">
  import MiniSearch from "minisearch";
  import DSSearchList from "$lib/components/DSSearchList.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import type { PageData } from "./$types";
  import type { WorkRecordSearchRow } from "$lib/svelte-types";

  let { data }: { data: PageData } = $props();

  type WorkRecordPrefix = WorkRecordSearchRow["prefix"];

  const prefixOptions: { id: WorkRecordPrefix; label: string }[] = [
    { id: "K", label: "K" },
    { id: "Q", label: "Q" },
    { id: "F", label: "F" },
  ];

  let selectedPrefix = $state<WorkRecordPrefix>("K");

  const index = $derived.by(() => {
    const itemsIndex = new MiniSearch<WorkRecordSearchRow>({
      idField: "work_record",
      fields: ["work_record", "search_text"],
      storeFields: ["work_record", "prefix", "entry_count", "search_text"],
      searchOptions: {
        combineWith: "AND",
        prefix: true,
        boost: { work_record: 4, search_text: 2 },
      },
    });

    itemsIndex.addAll(data.items);
    return itemsIndex;
  });

  const prefixFilter = $derived((item: WorkRecordSearchRow) => item.prefix === selectedPrefix);
</script>

<DSSearchList
  {index}
  filter={prefixFilter}
  inListHeader={selectedPrefix}
  fieldName="work_record"
  uiName="search work records..."
>
  {#snippet searchBarExtra()}
    <div class="flex items-center gap-2 max-[639px]:w-full max-[639px]:flex-wrap">
      <DSToggle
        bind:value={selectedPrefix}
        options={prefixOptions}
        ariaLabel="Work record prefix filter"
      />
    </div>
  {/snippet}

  {#snippet headline(item: WorkRecordSearchRow)}
    <a
      href={`/time/work-records/${encodeURIComponent(item.work_record)}`}
      class="text-blue-600 hover:underline"
    >
      {item.work_record}
    </a>
  {/snippet}

  {#snippet byline(item: WorkRecordSearchRow)}
    <span class="opacity-60">
      {item.entry_count} {item.entry_count === 1 ? "entry" : "entries"}
    </span>
  {/snippet}
</DSSearchList>
