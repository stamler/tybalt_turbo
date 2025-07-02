import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";
import type { ExpensesAugmentedResponse } from "$lib/pocketbase-types";

export const load: PageLoad = async ({ params }) => {
  try {
    const expense = await pb
      .collection("expenses_augmented")
      .getOne<ExpensesAugmentedResponse>(params.eid);

    return { expense };
  } catch (err) {
    console.error(`loading expense details: ${err}`);
    throw error(404, `Failed to load expense details`);
  }
};
