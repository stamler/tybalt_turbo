import { pb } from "$lib/pocketbase";
import { downloadCSV } from "$lib/utilities";

export async function downloadTimeEntryBranchMismatchesCsv() {
  await downloadCSV(
    `${pb.baseUrl}/api/reports/time_entry_branch_mismatches`,
    "time_entry_branch_mismatches.csv",
  );
}

export async function downloadActiveJobsCsv() {
  await downloadCSV(`${pb.baseUrl}/api/reports/active_jobs`, "active_jobs.csv");
}
