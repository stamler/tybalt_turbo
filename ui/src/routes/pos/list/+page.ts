import type { PurchaseOrdersResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ depends }) => {
  // Declare dependency on 'app:timeEntries'
  depends("app:purchaseOrders");

  let items: PurchaseOrdersResponse[];

  try {
    // load required data
    items = await pb.collection("purchase_orders").getFullList<PurchaseOrdersResponse>({
      expand: "division,job,type,uid.profiles_via_uid,approver,second_approver",
      sort: "-date",
    });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
