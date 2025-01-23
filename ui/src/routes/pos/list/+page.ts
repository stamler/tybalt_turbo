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
      // Note: ensure permissions are set to allow access to the related records.
      expand:
        "uid.profiles_via_uid,approver.profiles_via_uid,division,vendor,job,job.client,rejector.profiles_via_uid,category,second_approver.profiles_via_uid,second_approver_claim,parent_po",
      sort: "-date",
    });
    return {
      items,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
