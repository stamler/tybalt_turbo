import type { TimeEntriesResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { calculateTally } from "$lib/utilities";
import { error } from "@sveltejs/kit";

export const load: PageLoad = async ({ params }) => {
  try {
    const response = await pb.send(`/api/time_sheets/${params.id}/details`, {
      method: "GET",
    });

    const items = (response.items || []) as TimeEntriesResponse[];
    const tallies = calculateTally(items);
    const approverInfo = response.approverInfo || {
      approver_name: "",
      approved_date: "",
      committer_name: "",
      committed_date: "",
      rejector_name: "",
      rejected_date: "",
    };
    const committerInfo = {
      committer_name: approverInfo.committer_name || "",
    };

    return {
      items,
      tallies,
      timeSheet: response.timeSheet,
      timesheetId: params.id,
      approverInfo,
      committerInfo,
      sharedReviewerCount: response.sharedReviewerCount ?? 0,
    };
  } catch (err) {
    console.error(`loading time sheet details: ${err}`);
    throw error(404, `Failed to load time sheet details: ${err}`);
  }
};
