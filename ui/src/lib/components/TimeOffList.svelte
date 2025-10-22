<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import type { TimeOffResponse } from "$lib/pocketbase-types";

  let { items = [], header = "Time Off" }: { items?: TimeOffResponse[]; header?: string } =
    $props();
</script>

<DsList {items} inListHeader={header} search={true}>
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
