import type { TimeAmendmentsAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: TimeAmendmentsAugmentedResponse[];

  try {
    // load required data
    items = await pb.collection("time_amendments_augmented").getFullList<TimeAmendmentsAugmentedResponse>({
      sort: "-date",
    });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
