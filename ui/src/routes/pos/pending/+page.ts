import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
export const load: PageLoad = async () => {
  try {
    const result = await pb
      .collection("purchase_orders_augmented")
      .getFullList<PurchaseOrdersAugmentedResponse>({
        sort: "-date",
        // The rules should prevent visibility of Unapproved records by a user
        // who is not permitted to approve them.
        filter: pb.filter("status='Unapproved'"),
      });
    return {
      items: result,
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
