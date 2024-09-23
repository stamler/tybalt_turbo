import type { JobsResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: JobsResponse[];

  try {
    // load required data
    items = await pb.collection("jobs").getFullList({
      expand: "categories_via_job",
      sort: "-number",
    });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
