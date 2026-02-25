<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import type { ClaimHolder } from "./+page";

  let { data } = $props();
</script>

{#if data.error}
  <div class="p-4 text-red-600">{data.error}</div>
{:else if data.item}
  <div class="bg-neutral-100 px-3 py-2 text-sm text-neutral-600">
    {data.item.description}
  </div>
  {#if data.item.holders.length > 0}
    <DsList items={data.item.holders} search={true} inListHeader="{data.item.name} holders">
      {#snippet headline(holder: ClaimHolder)}
        <a
          href={`/admin_profiles/${holder.admin_profile_id}/details`}
          class="text-blue-600 hover:underline"
        >
          {holder.given_name}
          {holder.surname}
        </a>
      {/snippet}
    </DsList>
  {:else}
    <p class="p-4 text-sm text-neutral-500">No users hold this claim.</p>
  {/if}
{:else}
  <div class="p-4">Claim not found.</div>
{/if}
