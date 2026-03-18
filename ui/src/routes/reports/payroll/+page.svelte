<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { PageData } from "./$types";
  import type { PayrollReportWeekEndingsResponse } from "$lib/pocketbase-types";
  import { downloadCSV, downloadZip, shortDate } from "$lib/utilities";
  import { untrack } from "svelte";

  let { data }: { data: PageData } = $props();
  let weekEndings = $state(untrack(() => data.items));
  let receiptsLoading = $state(false);

  async function fetchTimeReport(weekEnding: string, week: number) {
    const url = `${pb.baseUrl}/api/reports/payroll_time/${weekEnding}/${week}`;
    const fileName = `payroll_time_report_${weekEnding}_week${week}.csv`;
    await downloadCSV(url, fileName);
  }

  async function fetchExpenseReport(payrollEnding: string) {
    const url = `${pb.baseUrl}/api/reports/payroll_expense/${payrollEnding}`;
    const fileName = `payroll_expense_report_${payrollEnding}.csv`;
    await downloadCSV(url, fileName);
  }

  async function fetchReceiptsReport(payrollEnding: string) {
    receiptsLoading = true;
    const url = `${pb.baseUrl}/api/reports/payroll_receipts/${payrollEnding}`;
    const fileName = `payroll_receipts_report_${payrollEnding}.zip`;
    await downloadZip(url, fileName);
    receiptsLoading = false;
  }
</script>

{#snippet anchor({ week_ending }: PayrollReportWeekEndingsResponse)}
  {shortDate(week_ending, true)}
{/snippet}
{#snippet headline({ committed_timesheet_count }: PayrollReportWeekEndingsResponse)}
  <span>{committed_timesheet_count} committed time sheet(s)</span>
{/snippet}
{#snippet line1({ committed_expense_count }: PayrollReportWeekEndingsResponse)}
  <span>{committed_expense_count} committed expense(s)</span>
{/snippet}
{#snippet actions({ week_ending }: PayrollReportWeekEndingsResponse)}
  <DsActionButton
    action={() => {
      fetchTimeReport(week_ending, 1);
    }}
    title="Week 1"
    color="orange">Week 1</DsActionButton
  >
  <DsActionButton
    action={() => {
      fetchTimeReport(week_ending, 2);
    }}
    title="Week 2"
    color="orange">Week 2</DsActionButton
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
    inListHeader="Payroll Reports"
    search={true}
    {anchor}
    {headline}
    {line1}
    {actions}
  />
{/if}
