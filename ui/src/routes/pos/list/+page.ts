import { pb } from "$lib/pocketbase";
import type { PurchaseOrdersAugmentedResponse, PurchaseOrdersResponse } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import { fetchVisiblePOs } from "$lib/poVisibility";
export const load: PageLoad = async () => {
  try {
    const currentUserId = pb.authStore.record?.id ?? "";
    const result = (await fetchVisiblePOs("mine")) as PurchaseOrdersAugmentedResponse[];
    return {
      items: result,
      createdItemIsVisible: (record: PurchaseOrdersResponse) => record.uid === currentUserId,
      realtime_source: "visible" as const,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
    return { items: [], realtime_source: "visible" as const };
  }
};
