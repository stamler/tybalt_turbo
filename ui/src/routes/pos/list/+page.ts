import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
export const load: PageLoad = async () => {
  try {
    // load all of the caller's own purchase orders
    const userId = get(authStore)?.model?.id || "";

    const result = await pb
      .collection("purchase_orders_augmented")
      .getFullList<PurchaseOrdersAugmentedResponse>({
        sort: "-date",
        filter: pb.filter("uid={:userId}", { userId }),
      });
    return {
      items: result,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
