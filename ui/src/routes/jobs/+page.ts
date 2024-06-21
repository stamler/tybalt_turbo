import type { JobsRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: JobsRecord[];

  try {
    // load required data
    items = await pb.collection("jobs").getFullList({
      // the - symbol means descending order
      sort: "-number",
    });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
