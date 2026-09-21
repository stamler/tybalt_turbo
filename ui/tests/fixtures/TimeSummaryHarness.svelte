<script lang="ts">
  import TimeSummaryContent from "../../src/lib/components/jobs/TimeSummaryContent.svelte";
  import type { SummaryKind } from "../../src/lib/components/jobs/timeSummary";
  let scenario = $state("normal");
  let kind = $state<SummaryKind>("staff");
  let startDate = $state("2026-09-01");
  const rateSheet = $derived(
    scenario === "no-sheet" ? undefined : { id: "sheet1", name: "Standard Rates", revision: 2 },
  );
</script>

<div class="mx-auto max-w-3xl py-6">
  <div class="mb-6 flex flex-wrap gap-4 px-4">
    <label
      >Scenario <select aria-label="Scenario" bind:value={scenario}>
        {#each ["normal", "mixed", "partial", "no-sheet", "unpriced", "empty", "error", "slow"] as option (option)}<option
            value={option}>{option}</option
          >{/each}
      </select></label
    >
    <label
      >Summary <select aria-label="Summary" bind:value={kind}
        ><option value="staff">Staff</option><option value="divisions">Divisions</option></select
      ></label
    >
    <label>Start date <input aria-label="Start date" type="date" bind:value={startDate} /></label>
  </div>
  <TimeSummaryContent jobId={scenario} {kind} {rateSheet} {startDate} endDate="2026-09-21" />
</div>
