<script lang="ts">
  import ObjectTable from "$lib/components/ObjectTable.svelte";
  import type { PageData } from "./$types";

  let { data }: { data: PageData } = $props();

  const tableData = $derived(
    data.items.map((item) => ({
      id: item.id,
      uid: item.uid,
      job_id: item.job_id,
      timesheet_id: item.timesheet_id,
      work_record: item.work_record,
      week_ending: item.week_ending,
      employee_name: [item.given_name, item.surname].filter(Boolean).join(" "),
      hours: item.hours,
      job_number: item.job_number,
      description: item.description,
    })),
  );

  const tableConfig = {
    columnFormatters: {},
    omitColumns: ["id", "uid", "job_id", "timesheet_id"],
    columnLabels: {
      work_record: "Work Record",
      week_ending: "Week Ending",
      employee_name: "Employee",
      hours: "Hours",
      job_number: "Job",
      description: "Description",
    },
    columnLinks: {
      week_ending: (row: Record<string, string>) =>
        row.timesheet_id ? `/time/sheets/${row.timesheet_id}/details` : null,
      job_number: (row: Record<string, string>) => (row.job_id ? `/jobs/${row.job_id}/details` : null),
    },
  };
</script>

<div class="mx-auto p-4">
  <h1 class="mb-4 text-2xl font-bold">Work Record {data.workRecord}</h1>
  <ObjectTable tableData={tableData} {tableConfig} />
</div>
