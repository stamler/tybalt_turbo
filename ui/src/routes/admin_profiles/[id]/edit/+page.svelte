<script lang="ts">
  import type { AdminProfilesEditPageData } from "$lib/svelte-types";
  import AdminProfilesEditor from "$lib/components/AdminProfilesEditor.svelte";
  let { data }: { data: AdminProfilesEditPageData } = $props();
</script>

<div class="mx-auto space-y-4 p-4">
  <h1 class="text-2xl font-bold">
    {#if data.item.surname || data.item.given_name}
      {data.item.given_name} {data.item.surname}
    {:else}
      {data.item.job_title || "-"}
    {/if}
  </h1>

  <AdminProfilesEditor {data} />
</div>
