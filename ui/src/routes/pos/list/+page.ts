import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: PurchaseOrdersAugmentedResponse[];

  try {
    // load augmented data, which contains comprehensive information about each
    // purchase order
    items = await pb
      .collection("purchase_orders_augmented")
      .getFullList<PurchaseOrdersAugmentedResponse>({
        sort: "-date",
      });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
