import type { ExpensesAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
export const load: PageLoad = async () => {
  try {
    // load all of the pending expenses for the caller
    const userId = get(authStore)?.model?.id || "";

    const result = await pb
      .collection("expenses_augmented")
      .getFullList<ExpensesAugmentedResponse>({
        sort: "-date",
        filter: pb.filter("approver={:approver} && approved='' && submitted=true", {
          approver: userId,
        }),
      });
    return {
      items: result,
      // createdItemIsVisible: (record: ExpensesResponse) => {
      // only show items where the caller is the approver. It should be
      // unnecessary to check whether the record is submitted because listRule
      // should prevent visibility of unsubmitted records by a user other than
      // the creator.
      // return record.approver === userId;
      // },
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
