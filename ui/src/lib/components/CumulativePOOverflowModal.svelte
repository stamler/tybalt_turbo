<script lang="ts">
  import DsActionButton from "./DSActionButton.svelte";
  import { fade } from "svelte/transition";
  import { createEventDispatcher } from "svelte";

  const dispatch = createEventDispatcher();

  let show = $state(false);
  let overflowData = $state<{
    parent_po: any;
    overflow_amount: number;
  } | null>(null);

  function closeModal() {
    show = false;
    overflowData = null;
  }

  export function openModal(data: { parent_po: any; overflow_amount: number }) {
    show = true;
    overflowData = data;
  }

  function handleCreateChild() {
    dispatch("createChild", overflowData);
    closeModal();
  }
</script>

{#if show}
  <div
    class="fixed inset-0 z-50 overflow-y-auto overflow-x-hidden"
    transition:fade={{ duration: 200 }}
  >
    <!-- Backdrop/overlay -->
    <div class="fixed inset-0 bg-black/80"></div>
    <!-- Modal content -->
    <div
      class="relative z-50 mx-auto my-20 flex w-11/12 flex-col rounded-lg bg-neutral-800 p-4 text-neutral-300 md:w-3/5"
    >
      <div class="flex items-start justify-between">
        <h1>Expense Exceeds Purchase Order Total</h1>
      </div>
      <div class="my-4 flex flex-col gap-4">
        {#if overflowData}
          <p>
            This expense exceeds the purchase order total by ${overflowData.overflow_amount.toFixed(
              2,
            )}.
          </p>
          <p>Would you like to create a child PO for the overflow amount?</p>
          <div class="flex gap-4">
            <DsActionButton action={handleCreateChild} color="green">
              Create Child PO
            </DsActionButton>
            <DsActionButton action={closeModal} color="red">Cancel</DsActionButton>
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
