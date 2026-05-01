<script lang="ts">
  import { resolve } from "$app/paths";
  import DsList from "$lib/components/DSList.svelte";
  import type { ClaimHolder } from "$lib/svelte-types";
  import Icon from "@iconify/svelte";

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
      {#snippet searchBarExtra()}
        <a
          href={resolve(`/claims/${data.item.id}/bulk_assign`)}
          class="flex items-center gap-1 rounded-sm bg-neutral-200 px-3 py-1 text-sm text-gray-700 hover:bg-neutral-300"
        >
          <Icon icon="mdi:edit-outline" class="text-base" /> Bulk Assign
        </a>
      {/snippet}
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
    <div class="flex items-center justify-between gap-3 p-4">
      <p class="text-sm text-neutral-500">No users hold this claim.</p>
      <a
        href={resolve(`/claims/${data.item.id}/bulk_assign`)}
        class="flex items-center gap-1 rounded-sm bg-neutral-200 px-3 py-1 text-sm text-gray-700 hover:bg-neutral-300"
      >
        <Icon icon="mdi:edit-outline" class="text-base" /> Bulk Assign
      </a>
    </div>
  {/if}
{:else}
  <div class="p-4">Claim not found.</div>
{/if}
