<script lang="ts">
  import { globalStore } from "$lib/stores/global";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DSPopover from "$lib/components/DSPopover.svelte";
  import { JobValue, valueCAD, valueDate } from "$lib/jobValue.svelte";
  // Actions beside the project value. The history list only includes changes
  // the signed-in user may see, so buttons appear only when they can work.
  let { value }: { value: JobValue } = $props();
  const claims = $derived($globalStore.claims);
  let showHistory = $state(false),
    confirmingDelete = $state(false);
  const latest = $derived(value.current);
  // Only the newest approval can be undone, and not while a change is open or
  // when it is the approval the project was created with.
  const canDeleteLatest = $derived(
    claims.includes("admin") && !!latest && !latest.initial_approval && !value.inProgress,
  );
  async function deleteLatest() {
    if (latest && (await value.act(latest, "delete"))) {
      confirmingDelete = false;
      showHistory = false;
    }
  }
</script>

<svelte:window onkeydown={(event) => event.key === "Escape" && (showHistory = false)} />

{#if value.current}
  <DsActionButton
    icon="mdi:file-download-outline"
    title="Download the authorization PDF for this value"
    color="blue"
    action={() => value.pdf(value.current!, true)}
  />
{/if}
{#if value.approved.length}
  <span class="relative">
    <DsActionButton
      icon="mdi:history"
      title="Value history"
      color="blue"
      action={() => (showHistory = !showHistory)}
    />
    {#if showHistory}
      <button
        type="button"
        aria-label="Close value history"
        class="fixed inset-0 z-40 cursor-default"
        onclick={() => (showHistory = false)}
      ></button>
      <div
        role="dialog"
        aria-label="Value history"
        class="absolute top-full left-0 z-50 mt-1 w-80 rounded-sm border border-neutral-300 bg-white p-3 shadow-lg"
      >
        <h3 class="mb-2 font-semibold">Value history</h3>
        <ol class="divide-y divide-neutral-200">
          {#each value.approved as item (item.id)}
            <li class="flex items-baseline justify-between gap-3 py-1.5 text-sm">
              <span class="font-semibold">{valueCAD(item.project_value)}</span>
              <span class="text-neutral-600"
                >{item.approver_name || item.approver} · {valueDate(item.approved)}</span
              >
            </li>
          {/each}
        </ol>
        {#if canDeleteLatest}
          <button
            type="button"
            class="mt-3 rounded bg-red-700 px-3 py-1 text-sm text-white hover:bg-red-800"
            onclick={() => (confirmingDelete = true)}>Delete latest approval</button
          >
        {/if}
      </div>
    {/if}
  </span>
{/if}
{#if claims.includes("job") && !value.inProgress && !value.loading}
  <DsActionButton color="yellow" action={() => value.openProposal(null)}
    >Propose new value</DsActionButton
  >
{/if}
{#if value.error && !confirmingDelete}<span role="alert" class="text-sm text-red-800"
    >{value.error}</span
  >{/if}

{#if latest}
  <DSPopover
    bind:show={confirmingDelete}
    title="Delete latest approval?"
    subtitle={`This deletes the approval of ${valueCAD(latest.project_value)} by ${latest.approver_name || latest.approver} on ${valueDate(latest.approved)}, and its PDF.`}
    error={value.error || null}
    submitting={value.busy}
    submitLabel={`Delete and restore ${valueCAD(latest.previous_project_value)}`}
    danger
    onSubmit={deleteLatest}
  >
    <p class="rounded-sm border border-red-300 bg-red-50 p-3 text-sm text-red-900">
      The project value returns to {valueCAD(latest.previous_project_value)}. This cannot be undone.
    </p>
  </DSPopover>
{/if}
