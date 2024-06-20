import type { JobsRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let jobs: JobsRecord[];

  try {
    // load required data
    jobs = await pb.collection("jobs").getFullList();
    return {
      jobs,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
