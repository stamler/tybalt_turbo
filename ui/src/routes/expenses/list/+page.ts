import type { ExpensesAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
export const load: PageLoad = async ({ depends }) => {
  // Declare dependency on 'app:expenses'
  depends("app:expenses");

  let items: ExpensesAugmentedResponse[];

  try {
    // load required data

    // TODO: this is only for the current user as loading all expenses is not
    // feasible. we must figure out two things: how to limit the number of
    // expenses loaded and how to handle messages that are visible by the caller
    // due to other permissions.
    const userId = get(authStore)?.model?.id || "";

    items = await pb
      .collection("expenses_augmented")
      .getFullList<ExpensesAugmentedResponse>({
        sort: "-date",
        filter: pb.filter("uid={:userId}", { userId }),
      });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
