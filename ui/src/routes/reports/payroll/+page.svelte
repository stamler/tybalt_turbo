<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { PageData } from "./$types";
  import type { TimeReportWeekEndingsResponse } from "$lib/pocketbase-types";
  import { shortDate, hoursWorked, jobs, divisions, payoutRequests } from "$lib/utilities";

  let { data }: { data: PageData } = $props();
  let weekEndings = $state(data.items);

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

{#snippet anchor({ week_ending }: TimeReportWeekEndingsResponse)}
  {week_ending}
{/snippet}
{#snippet headline()}
  Payroll
{/snippet}
{#snippet actions({ week_ending }: TimeReportWeekEndingsResponse)}
  Week 1
  <DsActionButton
    action={() => {
      fetchTimeReport(week_ending, 1);
    }}
    icon="mdi:download"
    title="Download"
    color="orange"
  />
{/snippet}

<!-- Show the list of items here -->
{#if weekEndings}
  <DsList items={weekEndings} search={true} {anchor} {headline} {actions} />
{/if}
