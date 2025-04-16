<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { globalStore } from "$lib/stores/global";
  import { shortDate, hoursWorked, jobs, divisions, payoutRequests } from "$lib/utilities";

  async function fetchTimeReport(weekEnding: string, week: number) {
    try {
      const timeReport = await pb.send(`/api/reports/payroll_time/${weekEnding}/${week}`, {
        method: "GET",
      });

      console.log(timeReport);
    } catch (error) {
      console.error("Error fetching time report:", error);
    }
  }
</script>

{#snippet anchor()}
  Date
{/snippet}
{#snippet headline()}
  Payroll
{/snippet}
{#snippet actions()}
  Week 1
  <DsActionButton
    action={() => {
      fetchTimeReport("2025-04-05", 1);
    }}
    icon="mdi:download"
    title="Download"
    color="orange"
  />
{/snippet}

<!-- Show the list of items here -->
<DsList items={$globalStore.time_sheets_tallies} search={true} {anchor} {headline} {actions} />
