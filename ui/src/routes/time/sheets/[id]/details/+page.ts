import type { TimeEntriesResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { calculateTally } from "$lib/utilities";
import { error } from "@sveltejs/kit";

export const load: PageLoad = async ({ params }) => {
  try {
    // Load time entries for the specific time sheet
    const items = await pb.collection("time_entries").getFullList<TimeEntriesResponse>({
      filter: pb.filter("tsid={:tsid}", { tsid: params.id }),
      expand: "job,time_type,division,category",
      sort: "-date",
    });

    // Calculate tallies for this specific time sheet
    const tallies = calculateTally(items);

    // Get the time sheet record for additional info
    const timeSheet = await pb.collection("time_sheets").getOne(params.id);

    // Get approver information via custom API endpoint
    let approverInfo = { approver_name: "", approved_date: "" };
    try {
      const approverResponse = await pb.send(`/api/time_sheets/${params.id}/approver`, {
        method: "GET",
      });
      approverInfo = approverResponse;
    } catch (err) {
      console.log("Could not fetch approver info:", err);
    }

    return {
      items,
      tallies,
      timeSheet,
      timesheetId: params.id,
      approverInfo,
    };
  } catch (err) {
    console.error(`loading time sheet details: ${err}`);
    throw error(404, `Failed to load time sheet details: ${err}`);
  }
}; 