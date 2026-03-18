import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import { fetchVisiblePOs } from "$lib/poVisibility";
import type { PageLoad } from "./$types";
export const load: PageLoad = async () => {
  const [pendingResult, approvedByMeResult] = await Promise.allSettled([
    pb.send("/api/purchase_orders/pending", {
      method: "GET",
    }) as Promise<PurchaseOrdersAugmentedResponse[]>,
    fetchVisiblePOs("approved_by_me_awaiting_second", undefined, 20),
  ]);

  if (pendingResult.status === "rejected") {
    console.error(`loading pending purchase orders: ${pendingResult.reason}`);
  }
  if (approvedByMeResult.status === "rejected") {
    console.error(`loading approved-by-me purchase orders: ${approvedByMeResult.reason}`);
  }

  return {
    // Keep the top section tied to actionable `/pending` semantics rather than
    // broader PO visibility. This remains the "can act now" queue.
    pendingData: {
      items: pendingResult.status === "fulfilled" ? pendingResult.value : [],
      realtime_source: "pending" as const,
    },
    approvedByMeAwaitingSecondData: {
      items: approvedByMeResult.status === "fulfilled" ? approvedByMeResult.value : [],
      realtime_source: "none" as const,
    },
  };
};
