<script lang="ts">
  import WIPContent from "../../src/lib/components/jobs/WIPContent.svelte";
  let scenario = $state("normal");
</script>

<div class="mx-auto max-w-4xl py-6">
  <label class="mb-4 block px-4"
    >Scenario
    <select aria-label="Scenario" bind:value={scenario}>
      {#each ["normal", "over", "partial", "unknown-po", "unknown-expense", "zero-project", "empty", "negative", "error", "slow"] as option (option)}
        <option value={option}>{option}</option>
      {/each}
    </select>
  </label>
  <WIPContent jobId={scenario} />
</div>
