<script lang="ts">
  import { globalStore } from "$lib/stores/global";

  function dismissError(id: string) {
    globalStore.dismissError(id);
  }
</script>

{#if $globalStore.errorMessages.length > 0}
  <div class="w-full bg-red-100 p-4">
    {#each $globalStore.errorMessages as error}
      <div class="mb-2 flex items-center justify-between text-red-800 last:mb-0">
        <span>{error.message}</span>
        <button
          class="ml-4 text-red-600 hover:text-red-800"
          on:click={() => dismissError(error.id)}
        >
          Dismiss
        </button>
      </div>
    {/each}
  </div>
{/if}
