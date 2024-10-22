import type { ExpensesResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ depends }) => {
  // Declare dependency on 'app:expenses'
  depends("app:expenses");

  let items: ExpensesResponse[];

  try {
    // load required data
    items = await pb.collection("expenses").getFullList<ExpensesResponse>({
      expand:
        "job,job.client,division,category,uid,approver,uid.profiles_via_uid,approver.profiles_via_uid,rejector.profiles_via_uid",
      sort: "-date",
    });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
