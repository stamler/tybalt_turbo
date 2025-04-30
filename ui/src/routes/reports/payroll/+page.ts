import type { TimeReportWeekEndingsResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: TimeReportWeekEndingsResponse[] = [];

  try {
    // load required data
    items = await pb.collection("time_report_week_endings").getFullList<TimeReportWeekEndingsResponse>();
    return {items};
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
