import type { TimeAmendmentsAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: TimeAmendmentsAugmentedResponse[];

  try {
    items = await pb
      .collection("time_amendments_augmented")
      .getFullList<TimeAmendmentsAugmentedResponse>({
        sort: "-date",
        filter: "committed_week_ending = ''",
      });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
