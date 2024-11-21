import type { TimeAmendmentsResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: TimeAmendmentsResponse[];

  try {
    // load required data
    items = await pb.collection("time_amendments").getFullList<TimeAmendmentsResponse>({
      expand: "uid,creator.profiles_via_uid,uid.profiles_via_uid,time_type,division,job,category",
      sort: "-date",
    });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
