import type { TimeTypesRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let timetypes: TimeTypesRecord[];

  try {
    // load required data
    timetypes = await pb.collection("time_types").getFullList({
      // the - symbol means descending order
      sort: "code",
    });
    return {
      timetypes,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
