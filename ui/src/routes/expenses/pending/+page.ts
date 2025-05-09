import type { ExpensesAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
export const load: PageLoad = async ({ depends }) => {
  // Declare dependency on 'app:expenses'
  depends("app:expenses");

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
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
