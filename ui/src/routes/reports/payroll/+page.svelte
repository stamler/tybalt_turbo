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

{#snippet anchor({ week_ending }: TimeReportWeekEndingsResponse)}
  {week_ending}
{/snippet}
{#snippet headline()}
  Payroll
{/snippet}
{#snippet actions({ week_ending }: TimeReportWeekEndingsResponse)}
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
    {actions}
  />
{/if}
