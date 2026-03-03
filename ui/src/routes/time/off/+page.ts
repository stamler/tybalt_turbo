import type { TimeOffResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import { authStore } from "$lib/stores/auth";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: TimeOffResponse[];

  try {
    const callerId = authStore.get()?.model?.id;
    const filter = callerId ? `id = "${callerId}" || manager_uid = "${callerId}"` : "";
    items = await pb.collection("time_off").getFullList({ filter });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
    return { items: [] };
  }
};
