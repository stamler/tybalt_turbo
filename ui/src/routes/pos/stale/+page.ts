import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
export const load: PageLoad = async () => {
  // 30 days ago as YYYY-MM-DD
  const staleDate = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().replace('T', ' ');
  try {
    const result = await pb
      .collection("purchase_orders_augmented")
      .getFullList<PurchaseOrdersAugmentedResponse>({
        sort: "-date",
        filter: pb.filter(
          "status='Active' && ((second_approval != '' && second_approval < '" + staleDate + "') || (approved < '" + staleDate + "'))"),
      });
    return {
      items: result,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
