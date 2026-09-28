<script lang="ts">
  import POApprovalLimits from "$lib/components/POApprovalLimits.svelte";
  import { globalStore } from "$lib/stores/global";
  import { canViewPOApprovers } from "$lib/poApprovalLimits";
</script>

<svelte:head><title>PO Approvers</title></svelte:head>

<main aria-label="PO Approvers">
  {#if canViewPOApprovers($globalStore.claims)}
    <POApprovalLimits />
  {:else}
    <p role="alert" class="rounded-sm border border-amber-200 bg-amber-50 p-4 text-amber-900">
      You need the report or po_approver claim to view PO approval limits.
    </p>
  {/if}
</main>
