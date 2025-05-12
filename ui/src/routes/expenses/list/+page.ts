import type { ExpensesAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
export const load: PageLoad = async () => {
  try {
    // load all of the caller's own expenses
    const userId = get(authStore)?.model?.id || "";

    const result = await pb
      .collection("expenses_augmented")
      .getFullList<ExpensesAugmentedResponse>({
        sort: "-date",
        filter: pb.filter("uid={:userId}", { userId }),
      });
    return {
      items: result,
      // createdItemIsVisible: (record: ExpensesResponse) => {
      //   return record.uid === userId;
      // },
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
