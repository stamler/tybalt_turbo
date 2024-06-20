<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import type { PageData } from "./$types";
  import type { TimeEntriesRecord } from "$lib/pocketbase-types";

  let { data }: { data: PageData } = $props();

  function hoursString(item: TimeEntriesRecord) {
    const hoursArray = [];
    if (item.hours) hoursArray.push(item.hours + " hrs");
    if (item.meals_hours) hoursArray.push(item.meals_hours + " hrs meals");
    return hoursArray.join(" + ");
  }
</script>

{#snippet anchor(item)}
  {item.date}
{/snippet}

{#snippet headline({ expand })}
  {#if expand?.time_type.code === "R"}
    <span>{expand.division.name}</span>
  {:else}
    <span>{expand?.time_type.name}</span>
  {/if}
{/snippet}

{#snippet byline({ expand, payout_request_amount })}
  {#if expand?.time_type.code === "OTO"}
  <span>${payout_request_amount}</span>
  {/if}
{/snippet}

{#snippet line1({ expand, job })}
  {#if ["R", "RT"].includes(expand?.time_type.code) && job !== ""}
    <span>{expand?.job.number}</span>
    {#if expand?.job.category}
      <span class="label">{expand.job.category}</span>
    {/if}
  {/if}
{/snippet}

{#snippet line2(item)}
  {hoursString(item)}
{/snippet}

{#snippet line3({ work_record, description})}
  {#if work_record !== ""}
    <span>Work Record: {work_record} / </span>
  {/if}
  <span>{description}</span>
{/snippet}

{#snippet actions({ id })}
  <a href="/details/{id}">details</a>
  <a href="/{id}">delete</a>
{/snippet}

<DsList items={data.items as TimeEntriesRecord[]} {anchor} {headline} {byline} {line1} {line2} {line3} {actions}/>
