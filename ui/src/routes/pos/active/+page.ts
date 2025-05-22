import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
export const load: PageLoad = async () => {
  try {
    const result = await pb
      .collection("purchase_orders_augmented")
      .getFullList<PurchaseOrdersAugmentedResponse>({
        sort: "-date",
        filter: pb.filter("status='Active'"),
      });
    return {
      items: result,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
