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
  import DsList from "$lib/components/DSList.svelte";
  import type { TimeOffResponse } from "$lib/pocketbase-types";
  import type { PageData } from "./$types";
  let { data }: { data: PageData } = $props();
</script>

<DsList items={data.items || []} inListHeader="Time Off" search={true}>
  {#snippet anchor(item: TimeOffResponse)}
    {item.opening_date}
  {/snippet}
  {#snippet headline(item: TimeOffResponse)}
    {item.name}
  {/snippet}
  {#snippet line1(item: TimeOffResponse)}
    <span class="opacity-30">PPTO Available</span>
    {item.opening_op - item.used_op}
    <span class="opacity-30">Opening</span>
    {item.opening_op}
    <span class="opacity-30">Used</span>
    {item.used_op}
    <span class="opacity-30">({item.timesheet_op} on timesheets)</span>
    <span class="opacity-30">As of</span>
    {item.last_op}
  {/snippet}
  {#snippet line2(item: TimeOffResponse)}
    <span class="opacity-30">Vacation Available</span>
    {item.opening_ov - item.used_ov}
    <span class="opacity-30">Opening</span>
    {item.opening_ov}
    <span class="opacity-30">Used</span>
    {item.used_ov}
    <span class="opacity-30">({item.timesheet_ov} on timesheets)</span>
    <span class="opacity-30">As of</span>
    {item.last_ov}
  {/snippet}
</DsList>
