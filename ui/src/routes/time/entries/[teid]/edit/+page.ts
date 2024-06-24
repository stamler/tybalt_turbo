import type { TimeEntriesRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ params }) => {
  let item: TimeEntriesRecord;
  try {
    item = await pb.collection("time_entries").getOne(params.teid);
    return { item };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
