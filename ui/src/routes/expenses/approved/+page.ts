import type { ExpensesAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
export const load: PageLoad = async () => {
  try {
    // load the last 50 expenses the caller has approved
    const userId = get(authStore)?.model?.id || "";

    const result = await pb
      .collection("expenses_augmented")
      .getList<ExpensesAugmentedResponse>(1, 50, {
        sort: "-date",
        filter: pb.filter("approver={:approver} && approved!=''", {
          approver: userId,
        }),
      });
    return {
      items: result.items,
      // createdItemIsVisible: (record: ExpensesResponse) => {
      //   return record.approved !== "" && record.approver === userId;
      // },
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
