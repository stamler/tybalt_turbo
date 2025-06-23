<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { PageData } from "./$types";
  import type { TimeTrackingResponse } from "$lib/pocketbase-types";
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

  async function fetchTimeSummaryByEmployee(week_ending: string) {
    const url = `${pb.baseUrl}/api/reports/weekly_time_by_employee/${week_ending}`;
    const fileName = `weekly_time_by_employee_${week_ending}.csv`;
    await downloadCSV(url, fileName);
  }
</script>

{#snippet anchor({ week_ending }: TimeTrackingResponse)}
  {week_ending}
{/snippet}
{#snippet headline({ committed_count }: TimeTrackingResponse)}
  {#if committed_count > 0}
    {committed_count} committed time sheet(s)
  {/if}
{/snippet}
{#snippet line1({ approved_count }: TimeTrackingResponse)}
  {#if approved_count > 0}
    {approved_count} approved
  {/if}
{/snippet}
{#snippet line3({ submitted_count }: TimeTrackingResponse)}
  {#if submitted_count > 0}
    {submitted_count} submitted & pending approval
  {/if}
{/snippet}
{#snippet actions({ week_ending }: TimeTrackingResponse)}
  <DsActionButton
    action={() => {
      fetchTimeReport(week_ending);
    }}
    title="Time Report"
    color="orange">Time</DsActionButton
  >
  <DsActionButton
    action={() => {
      fetchTimeSummaryByEmployee(week_ending);
    }}
    title="Time Summary by Employee"
    color="orange">Time Summary</DsActionButton
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
    {line1}
    {line3}
    {actions}
  />
{/if}
