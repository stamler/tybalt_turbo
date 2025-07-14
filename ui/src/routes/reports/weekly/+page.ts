import type { TimeTrackingResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: TimeTrackingResponse[] = [];

  try {
    // load required data
    items = await pb.collection("time_tracking").getFullList<TimeTrackingResponse>();
    return { items };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
