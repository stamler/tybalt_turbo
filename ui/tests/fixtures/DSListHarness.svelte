<script lang="ts">
  import DSList from "$lib/components/DSList.svelte";
  import rows from "./dsList.json";
  let state = $state({ searchTerm: "", page: 1, perPage: 10 });
</script>

<section aria-label="Paged list">
  <DSList
    items={rows}
    search
    pagination
    searchTerm={state.searchTerm}
    page={state.page}
    perPage={state.perPage}
    searchPlaceholder="Filter contacts"
    searchText={(item) => `${item.name} ${item.email}`}
    onStateChange={(next) => (state = next)}
  >
    {#snippet headline(item)}<span data-row>{item.name}</span>{/snippet}
  </DSList>
</section>
<section aria-label="Default list">
  <DSList items={rows} search>
    {#snippet headline(item)}<span data-row>{item.name}</span>{/snippet}
  </DSList>
</section>
<section aria-label="Grouped list">
  <DSList items={rows} search groupField="group" groupSort="ASC">
    {#snippet headline(item)}<span data-row>{item.name}</span>{/snippet}
  </DSList>
</section>
