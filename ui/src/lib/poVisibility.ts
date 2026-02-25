import { pb } from "$lib/pocketbase";
import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";

export type VisiblePOScope = "all" | "mine" | "active" | "rejected" | "stale" | "expiring";

export async function fetchVisiblePOs(
  scope: VisiblePOScope,
  beforeDate?: string,
): Promise<PurchaseOrdersAugmentedResponse[]> {
  const params = new URLSearchParams({ scope });
  if (scope === "stale" && beforeDate) {
    params.set("stale_before", beforeDate);
  }
  if (scope === "expiring" && beforeDate) {
    params.set("expiring_before", beforeDate);
  }

  return (await pb.send(`/api/purchase_orders/visible?${params.toString()}`, {
    method: "GET",
  })) as PurchaseOrdersAugmentedResponse[];
}

export async function fetchVisiblePO(id: string): Promise<PurchaseOrdersAugmentedResponse> {
  return (await pb.send(`/api/purchase_orders/visible/${id}`, {
    method: "GET",
  })) as PurchaseOrdersAugmentedResponse;
}

export async function fetchPendingPO(id: string): Promise<PurchaseOrdersAugmentedResponse> {
  return (await pb.send(`/api/purchase_orders/pending/${id}`, {
    method: "GET",
  })) as PurchaseOrdersAugmentedResponse;
}
