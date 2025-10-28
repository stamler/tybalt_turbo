<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import { authStore } from "$lib/stores/auth";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { shortDate, downloadCSV, downloadZip } from "$lib/utilities";

  type Row = {
    id: string; // DSList requires id
    pay_period_ending: string;
    submitted_count: number;
    approved_count: number;
    committed_count: number;
    rejected_count: number;
  };

  let rows: Row[] = [];

  async function init() {
    // Only attempt to load data if user is authenticated
    if (!$authStore?.isValid) {
      return;
    }

    try {
      const res: Omit<Row, "id">[] = await pb.send("/api/expenses/tracking_counts", {
        method: "GET",
      });
      rows = res.map((r) => ({ ...r, id: r.pay_period_ending }));
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to load tracking counts");
    }
  }

  init();

  function openWeek(ppe: string) {
    window.location.href = `/expenses/tracking/${ppe}`;
  }

  async function fetchExpenseReport(ppe: string) {
    const url = `${pb.baseUrl}/api/reports/payroll_expense/${ppe}`;
    const fileName = `payroll_expense_report_${ppe}.csv`;
    await downloadCSV(url, fileName);
  }

  async function fetchReceiptsReport(ppe: string) {
    const url = `${pb.baseUrl}/api/reports/payroll_receipts/${ppe}`;
    const fileName = `payroll_receipts_report_${ppe}.zip`;
    await downloadZip(url, fileName);
  }
</script>

<DsList items={rows} inListHeader="Expenses Tracking">
  {#snippet anchor({ id, pay_period_ending }: Row)}
    <a class="font-bold hover:underline" href={`/expenses/tracking/${id}`}
      >{shortDate(pay_period_ending, true)}</a
    >
  {/snippet}
  {#snippet headline(r: Row)}
    <div class="flex items-center gap-4">
      {#if r.approved_count > 0}
        <span>Approved: {r.approved_count}</span>
      {/if}
    </div>
  {/snippet}
  {#snippet line1(r: Row)}
    {#if r.committed_count > 0}
      <span>{r.committed_count} committed expense(s)</span>
    {/if}
  {/snippet}
  {#snippet line2(r: Row)}
    {#if r.submitted_count > 0}
      <span>Submitted: {r.submitted_count}</span>
    {/if}
    {#if r.rejected_count > 0}
      <span>Rejected: {r.rejected_count}</span>
    {/if}
  {/snippet}
  {#snippet actions(r: Row)}
    <DsActionButton action={() => fetchExpenseReport(r.pay_period_ending)} title="Expense Report"
      >Expenses</DsActionButton
    >
    <DsActionButton action={() => fetchReceiptsReport(r.pay_period_ending)} title="Receipts Archive"
      >Receipts</DsActionButton
    >
  {/snippet}
</DsList>
