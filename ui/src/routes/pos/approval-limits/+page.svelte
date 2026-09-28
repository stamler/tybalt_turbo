<script lang="ts">
  import POApprovalLimits from "$lib/components/POApprovalLimits.svelte";
  import { globalStore } from "$lib/stores/global";
</script>

<svelte:head><title>PO Approvers</title></svelte:head>

<main aria-label="PO Approvers">
  {#if $globalStore.claims.includes("report")}
    <POApprovalLimits />
  {:else}
    <p role="alert" class="rounded-sm border border-amber-200 bg-amber-50 p-4 text-amber-900">
      You need the report claim to view PO approval limits.
    </p>
  {/if}
</main>
