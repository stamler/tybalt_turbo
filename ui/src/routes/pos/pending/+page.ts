import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
export const load: PageLoad = async () => {
  try {
    const result = (await pb.send("/api/purchase_orders/pending", {
      method: "GET",
    })) as PurchaseOrdersAugmentedResponse[];
    return {
      items: result,
      realtime_source: "pending" as const,
      // createdItemIsVisible: (record: PurchaseOrdersResponse) => {
      // only show items where the caller is the approver. It should be
      // unnecessary to check whether the record is submitted because listRule
      // should prevent visibility of unsubmitted records by a user other than
      // the creator.
      // return record.approver === userId;
      // },
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
