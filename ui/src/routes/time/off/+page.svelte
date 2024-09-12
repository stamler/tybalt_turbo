<script lang="ts">
  // The time off page shows the contents of the time_off collection. Because
  // the api rules only allow the viewing of time_off records they are allowed
  // to see, we can simply fetch all the records and display them.

  // We will depend on a component that we'll recreate in svelte for the display
  // of the table. I used ObjectTable.vue in the past for tybalt, and this was
  // reimplemented and updated in the stanalytics project. That will need to be
  // updated to svelte as well. Once that is done, we'll just display the data
  // in an ObjectTable component and it will be done.

  // Because time_off is a view collection, it cannot be updated so there's no
  // need to implement any reactive statements. We can just fetch the data once
  // and use that result.
  import ObjectTable from "$lib/components/ObjectTable.svelte";
  import type { PageData } from "./$types";
  let { data }: { data: PageData } = $props();
</script>

<ObjectTable
  tableData={data.items || []}
  tableConfig={{
    omitColumns: ["collectionId", "collectionName", "createdBy", "createdAt", "updatedAt"],
    columnFormatters: {
      type: (value) => {
        return value;
      },
    },
  }}
>
  {#snippet rowActions(id)}
    {id}
  {/snippet}
</ObjectTable>
