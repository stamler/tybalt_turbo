<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import type { ClaimsResponse } from "$lib/pocketbase-types";

  let { data } = $props();
  let items = $derived(data.items);
</script>

<DsList {items} search={true}>
  {#snippet headline(item: ClaimsResponse)}
    <a href={`/claims/${item.id}/details`} class="text-blue-600 hover:underline">
      {item.name}
    </a>
  {/snippet}

  {#snippet line3(item: ClaimsResponse)}
    {item.description}
  {/snippet}
</DsList>
