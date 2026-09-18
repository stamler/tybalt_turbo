import type { PayrollReportWeekEndingsResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: PayrollReportWeekEndingsResponse[] = [];

  try {
    // load required data
    items = await pb
      .collection("payroll_report_week_endings")
      .getFullList<PayrollReportWeekEndingsResponse>({ sort: "-week_ending" });
    return { items };
  } catch (error) {
    console.error(`loading data: ${error}`);
    return { items: [] };
  }
};
