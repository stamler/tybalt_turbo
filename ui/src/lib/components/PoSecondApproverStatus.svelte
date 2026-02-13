<script lang="ts">
  import type { SecondApproverStatus, SecondApproversResponse } from "$lib/svelte-types";

  // Shared rendering for second-approver status messaging + diagnostics.
  // This keeps the PO editor focused on data flow while this component owns
  // display states (errors, status hints, and optional "why" details).
  let {
    showFetchError = false,
    showStatusHint = false,
    status = "",
    reasonMessage = "",
    meta = null,
    division = "",
    kindLabel = "",
    hasJob = false,
  }: {
    showFetchError?: boolean;
    showStatusHint?: boolean;
    status?: SecondApproverStatus | "";
    reasonMessage?: string;
    meta?: SecondApproversResponse["meta"] | null;
    division?: string;
    kindLabel?: string;
    hasJob?: boolean;
  } = $props();

  let showWhy = $state(false);

  const formatAmount = (value: number) =>
    Number.isFinite(value) ? value.toFixed(2) : String(value);

  // Collapse diagnostics whenever status/meta changes so users do not keep stale
  // details open after editing fields that trigger a new eligibility evaluation.
  $effect(() => {
    status;
    meta;
    showWhy = false;
  });
</script>

{#if showFetchError}
  <span class="flex w-full gap-2 text-sm text-red-600">
    <span class="invisible">Priority Second Approver</span>
    <span>Unable to load approver options right now. Please try again.</span>
  </span>
{:else if showStatusHint && status === "not_required"}
  <span class="flex w-full gap-2 text-sm text-neutral-600">
    <span class="invisible">Priority Second Approver</span>
    <span>Second approver is not required for this purchase order.</span>
  </span>
{:else if showStatusHint && status === "requester_qualifies"}
  <span class="flex w-full gap-2 text-sm text-neutral-600">
    <span class="invisible">Priority Second Approver</span>
    <span>Second approval is required; you qualify and will be assigned automatically.</span>
  </span>
{:else if showStatusHint && status === "required_no_candidates"}
  <span class="flex w-full gap-2 text-sm text-red-600">
    <span class="invisible">Priority Second Approver</span>
    <span>
      {reasonMessage ||
        "Second approval is required, but no eligible second approver is available."}
    </span>
  </span>
{/if}

{#if showStatusHint && meta}
  <span class="flex w-full gap-2 text-sm">
    <span class="invisible">Priority Second Approver</span>
    <button
      type="button"
      class="text-neutral-600 underline hover:text-neutral-900"
      onclick={() => {
        showWhy = !showWhy;
      }}
    >
      {showWhy ? "Hide why" : "Why?"}
    </button>
  </span>
  {#if showWhy}
    <span class="flex w-full gap-2 text-xs text-neutral-600">
      <span class="invisible">Priority Second Approver</span>
      <span class="flex flex-col">
        <span>reason code: {meta.reason_code || "n/a"}</span>
        <span>evaluated amount: ${formatAmount(meta.evaluated_amount)}</span>
        <span>second-approval threshold: ${formatAmount(meta.second_approval_threshold)}</span>
        <span>tier ceiling: ${formatAmount(meta.tier_ceiling)}</span>
        <span>eligibility limit rule: {meta.limit_column || "n/a"}</span>
        <span>division: {division || "n/a"}</span>
        <span>kind: {kindLabel || "n/a"}</span>
        <span>has job: {hasJob ? "yes" : "no"}</span>
      </span>
    </span>
  {/if}
{/if}
