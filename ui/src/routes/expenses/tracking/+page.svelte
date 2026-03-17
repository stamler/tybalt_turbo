<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import { authStore } from "$lib/stores/auth";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { shortDate, downloadCSV, downloadZip } from "$lib/utilities";

  type Row = {
    id: string; // DSList requires id
    committed_week_ending: string;
    committed_count: number;
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
      rows = res.map((r) => ({ ...r, id: r.committed_week_ending }));
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to load tracking counts");
    }
  }

  init();

  async function fetchExpenseReport(cwe: string) {
    const url = `${pb.baseUrl}/api/reports/weekly_expense/${cwe}`;
    const fileName = `expense_report_${cwe}.csv`;
    await downloadCSV(url, fileName);
  }

  async function fetchReceiptsReport(cwe: string) {
    const url = `${pb.baseUrl}/api/reports/weekly_receipts/${cwe}`;
    const fileName = `receipts_report_${cwe}.zip`;
    await downloadZip(url, fileName);
  }
</script>

<DsList items={rows} inListHeader="Expenses Tracking">
  {#snippet anchor({ id, committed_week_ending }: Row)}
    <a class="font-bold hover:underline" href={`/expenses/tracking/${id}`}
      >{shortDate(committed_week_ending, true)}</a
    >
  {/snippet}
  {#snippet headline(r: Row)}
    <span>{r.committed_count} committed expense(s)</span>
  {/snippet}
  {#snippet actions(r: Row)}
    <DsActionButton action={() => fetchExpenseReport(r.committed_week_ending)} title="Expense Report"
      >Expenses</DsActionButton
    >
    <DsActionButton action={() => fetchReceiptsReport(r.committed_week_ending)} title="Receipts Archive"
      >Receipts</DsActionButton
    >
  {/snippet}
</DsList>
