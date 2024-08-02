import type { TimeEntriesResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ depends }) => {
  // Declare dependency on 'app:timeEntries'
  depends("app:timeEntries");

  let items: TimeEntriesResponse[];

  try {
    // load required data
    items = await pb.collection("time_entries").getFullList<TimeEntriesResponse>({
      filter: pb.filter('tsid=""'),
      expand: "job,time_type,division",
      sort: "-date",
    });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
