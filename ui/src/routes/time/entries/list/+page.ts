import type { TimeEntriesResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";

export const load: PageLoad = async () => {
  try {
    // load only the current caller's entries (uid match) that are not yet
    // bundled into a time-sheet (tsid="").
    const userId = get(authStore)?.model?.id || "";

    const items = await pb.collection("time_entries").getFullList<TimeEntriesResponse>({
      filter: pb.filter("uid={:userId} && tsid=''", { userId }),
      expand: "job,time_type,division,category",
      sort: "-date",
    });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
