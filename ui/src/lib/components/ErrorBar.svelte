<script lang="ts">
  import { globalStore } from "$lib/stores/global";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  function dismissError(id: string) {
    globalStore.dismissError(id);
  }
</script>

{#if $globalStore.errorMessages.length > 0}
  <div class="w-full bg-red-100 p-4">
    {#each $globalStore.errorMessages as error}
      <div class="mb-2 flex items-center justify-between text-red-800 last:mb-0">
        <span>{error.message}</span>
        <DsActionButton action={() => dismissError(error.id)} color="red">Dismiss</DsActionButton>
      </div>
    {/each}
  </div>
{/if}
