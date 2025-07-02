import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";
import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";

export const load: PageLoad = async ({ params }) => {
  try {
    const po = await pb
      .collection("purchase_orders_augmented")
      .getOne<PurchaseOrdersAugmentedResponse>(params.poid);

    // fetch related expenses
    const expenses = await pb.collection("expenses_augmented").getFullList({
      filter: `purchase_order='${params.poid}'`,
      sort: "-date",
    });

    return { po, expenses };
  } catch (err) {
    console.error(`loading purchase order details: ${err}`);
    throw error(404, `Failed to load purchase order details`);
  }
};
