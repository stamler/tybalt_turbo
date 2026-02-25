<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import type { ClaimListItem } from "$lib/svelte-types";

  let { data } = $props();
  let items: ClaimListItem[] = $derived(data.items);
</script>

<DsList {items} search={true}>
  {#snippet headline(item: ClaimListItem)}
    <a href={`/claims/${item.id}/details`} class="text-blue-600 hover:underline">
      {item.name}
    </a>
  {/snippet}

  {#snippet byline(item: ClaimListItem)}
    <span class="opacity-60">
      {item.holder_count} {item.holder_count === 1 ? "holder" : "holders"}
    </span>
  {/snippet}

  {#snippet line3(item: ClaimListItem)}
    {item.description}
  {/snippet}
</DsList>
