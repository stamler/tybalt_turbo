import type { PurchaseOrdersAugmentedResponse, PurchaseOrdersResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let items: PurchaseOrdersResponse[];
  let augmentedItems: PurchaseOrdersAugmentedResponse[];

  try {
    // load augmented data, which contains extra calculated columns about each
    // purchase order
    augmentedItems = await pb.collection("purchase_orders_augmented").getFullList<PurchaseOrdersAugmentedResponse>();

    // load required data
    items = await pb.collection("purchase_orders").getFullList<PurchaseOrdersResponse>({
      // TODO: join with lower and upper thresholds to allow UI to show or hide "approve", "reject" buttons
      // TODO: store whether the user has the payables_admin claim somewhere in the app then show the "cancel" button if the status is "Active" and there are no expenses associated with the PO
      // TODO: show the close button if the user has the "payables_admin" claim, the status is "Active", type is not "Normal" and there are expenses associated with the PO

      // For all three of these, it may be useful to use the
      // purchase_orders_augmented view to determine whether to show the
      // buttons. We could also update the listRule and viewRule of this view to
      // restrict visibility of the augmented colums to only the users who have
      // the necessary permissions.

      // Finally, every time a realtime event occurs on the purchase_orders
      // collection, we must reload the purchase_orders_augmented view to
      // reflect changes.

      // Alternatively, all of this can be done on the client side using only
      // the user's claims and knowledge of the thresholds below and above their
      // max_amount
      expand:
        "uid.profiles_via_uid,approver.profiles_via_uid,division,vendor,job,job.client,rejector.profiles_via_uid,category,second_approver.profiles_via_uid,second_approver_claim,parent_po",
      sort: "-date",
    });
    return {
      items,
      augmentedItems,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
