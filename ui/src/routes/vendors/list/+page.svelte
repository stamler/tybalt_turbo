<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { globalStore } from "$lib/stores/global";
  import type { VendorsResponse } from "$lib/pocketbase-types";

  let items = $state($globalStore.vendors);
</script>

<DsList {items} search={true}>
  {#snippet headline({ name, alias }: VendorsResponse)}
    <span class="flex items-center gap-2">
      {name} ({alias})
    </span>
  {/snippet}

  {#snippet actions({ id }: VendorsResponse)}
    <DsActionButton
      action={`/vendors/${id}/edit`}
      icon="mdi:edit-outline"
      title="Edit"
      color="blue"
    />
    <DsActionButton
      action={() => globalStore.deleteItem("vendors", id)}
      icon="mdi:delete"
      title="Delete"
      color="red"
    />
  {/snippet}
</DsList>
