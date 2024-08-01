import type { TimeEntriesRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ depends }) => {
  // Declare dependency on 'app:timeEntries'
  depends('app:timeEntries');

  let items: TimeEntriesRecord[];

  try {
    // load required data
    items = await pb.collection("time_entries").getFullList({
      filter: pb.filter("tsid=\"\""),
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