import { pb } from "$lib/pocketbase";
import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";

export type VisiblePOScope =
  | "all"
  | "mine"
  | "active"
  | "rejected"
  | "stale"
  | "expiring"
  | "approved_by_me_awaiting_second";

export type VisiblePurchaseOrderResponse = PurchaseOrdersAugmentedResponse & {
  covered_within_project_budget: boolean;
  expenses_total: number;
  has_project_authorization: boolean;
  recurring_expected_occurrences: number;
  recurring_remaining_occurrences: number;
  remaining_amount: number;
};

export async function fetchVisiblePOs(
  scope: VisiblePOScope,
  beforeDate?: string,
  limit?: number,
): Promise<VisiblePurchaseOrderResponse[]> {
  const params = new URLSearchParams({ scope });
  if (scope === "stale" && beforeDate) {
    params.set("stale_before", beforeDate);
  }
  if (scope === "expiring" && beforeDate) {
    params.set("expiring_before", beforeDate);
  }
  if (limit !== undefined) {
    params.set("limit", String(limit));
  }

  return (await pb.send(`/api/purchase_orders/visible?${params.toString()}`, {
    method: "GET",
  })) as VisiblePurchaseOrderResponse[];
}

export async function fetchVisiblePO(id: string): Promise<VisiblePurchaseOrderResponse> {
  return (await pb.send(`/api/purchase_orders/visible/${id}`, {
    method: "GET",
  })) as VisiblePurchaseOrderResponse;
}

export async function fetchPendingPO(id: string): Promise<VisiblePurchaseOrderResponse> {
  return (await pb.send(`/api/purchase_orders/pending/${id}`, {
    method: "GET",
  })) as VisiblePurchaseOrderResponse;
}
