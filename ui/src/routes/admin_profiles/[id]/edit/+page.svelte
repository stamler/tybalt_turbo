<script lang="ts">
  import type { AdminProfilesPageData } from "$lib/svelte-types";
  import AdminProfilesEditor from "$lib/components/AdminProfilesEditor.svelte";
  let { data }: { data: AdminProfilesPageData } = $props();
</script>

<div class="mx-auto space-y-4 p-4">
  <h1 class="text-2xl font-bold">
    {#if (data.item as any).surname || (data.item as any).given_name}
      {(data.item as any).given_name} {(data.item as any).surname}
    {:else}
      {data.item.job_title || "-"}
    {/if}
  </h1>

  <AdminProfilesEditor {data} />
</div>
