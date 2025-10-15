<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { shortDate, downloadCSV } from "$lib/utilities";
  import { goto } from "$app/navigation";

  type Row = {
    id: string; // DSList requires id
    week_ending: string;
    submitted_count: number;
    approved_count: number;
    committed_count: number;
    rejected_count: number;
  };

  let rows: Row[] = [];

  async function init() {
    try {
      const res: Omit<Row, "id">[] = await pb.send("/api/time_sheets/tracking_counts", {
        method: "GET",
      });
      rows = res.map((r) => ({ ...r, id: r.week_ending }));
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to load tracking counts");
    }
  }

  init();

  function openWeek(weekEnding: string) {
    goto(`/time/tracking/${weekEnding}`);
  }

  async function downloadWeeklyTime(weekEnding: string) {
    const url = `${pb.baseUrl}/api/reports/weekly_time/${weekEnding}`;
    const fileName = `weekly_time_report_${weekEnding}.csv`;
    await downloadCSV(url, fileName);
  }

  async function downloadWeeklyByEmployee(weekEnding: string) {
    const url = `${pb.baseUrl}/api/reports/weekly_time_by_employee/${weekEnding}`;
    const fileName = `weekly_time_by_employee_${weekEnding}.csv`;
    await downloadCSV(url, fileName);
  }
</script>

<DsList items={rows} inListHeader="Time Tracking">
  {#snippet anchor({ id, week_ending }: Row)}
    <a class="font-bold hover:underline" href={`/time/tracking/${id}`}
      >{shortDate(week_ending, true)}</a
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
      <span>{r.committed_count} committed time sheet(s)</span>
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
    <DsActionButton action={() => downloadWeeklyTime(r.week_ending)}>Weekly Time</DsActionButton>
    <DsActionButton action={() => downloadWeeklyByEmployee(r.week_ending)}
      >By Employee</DsActionButton
    >
  {/snippet}
</DsList>
