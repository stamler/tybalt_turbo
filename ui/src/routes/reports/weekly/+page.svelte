<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { PageData } from "./$types";
  import type { TimeReportWeekEndingsResponse } from "$lib/pocketbase-types";
  import { downloadCSV, downloadZip } from "$lib/utilities";

  let { data }: { data: PageData } = $props();
  let weekEndings = $state(data.items);
  let receiptsLoading = $state(false);

  async function fetchTimeReport(weekEnding: string) {
    const url = `${pb.baseUrl}/api/reports/weekly_time/${weekEnding}`;
    const fileName = `weekly_time_report_${weekEnding}.csv`;
    await downloadCSV(url, fileName);
  }

  async function fetchExpenseReport(weekEnding: string) {
    const url = `${pb.baseUrl}/api/reports/weekly_expense/${weekEnding}`;
    const fileName = `weekly_expense_report_${weekEnding}.csv`;
    await downloadCSV(url, fileName);
  }

  async function fetchReceiptsReport(weekEnding: string) {
    receiptsLoading = true;
    const url = `${pb.baseUrl}/api/reports/weekly_receipts/${weekEnding}`;
    const fileName = `weekly_receipts_report_${weekEnding}.zip`;
    await downloadZip(url, fileName);
    receiptsLoading = false;
  }
</script>

{#snippet anchor({ week_ending }: TimeReportWeekEndingsResponse)}
  {week_ending}
{/snippet}
{#snippet headline()}
  Weekly
{/snippet}
{#snippet actions({ week_ending }: TimeReportWeekEndingsResponse)}
  <DsActionButton
    action={() => {
      fetchTimeReport(week_ending);
    }}
    title="Time Report"
    color="orange">Time</DsActionButton
  >
  <DsActionButton
    action={() => {
      fetchExpenseReport(week_ending);
    }}
    title="Expense Report"
    color="orange">Expenses</DsActionButton
  >
  <DsActionButton
    action={() => {
      fetchReceiptsReport(week_ending);
    }}
    loading={receiptsLoading}
    title="Receipts Archive"
    color="orange">Receipts</DsActionButton
  >
{/snippet}

<!-- Show the list of items here -->
{#if weekEndings}
  <DsList
    items={weekEndings}
    inListHeader="Weekly Reports"
    search={true}
    {anchor}
    {headline}
    {actions}
  />
{/if}
