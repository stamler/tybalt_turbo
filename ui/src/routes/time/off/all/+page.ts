import type { TimeOffResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: TimeOffResponse[];

  try {
    items = await pb.collection("time_off").getFullList();
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
