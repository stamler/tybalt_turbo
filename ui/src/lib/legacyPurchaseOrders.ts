import { pb } from "$lib/pocketbase";
import type { PurchaseOrdersRecord, PurchaseOrdersResponse } from "$lib/pocketbase-types";

type LegacyPurchaseOrderLike = PurchaseOrdersRecord | PurchaseOrdersResponse;

export function buildLegacyPurchaseOrderPayload(record: LegacyPurchaseOrderLike): Record<string, unknown> {
  return {
    uid: record.uid,
    approver: record.approver,
    po_number: record.po_number,
    date: record.date,
    division: record.division,
    description: record.description,
    payment_type: record.payment_type,
    total: record.total,
    vendor: record.vendor,
    type: record.type,
    kind: record.kind,
    branch: record.branch,
    job: record.job,
  };
}

export async function fetchLegacyPurchaseOrderForEdit(id: string): Promise<PurchaseOrdersResponse> {
  return (await pb.send(`/api/purchase_orders/legacy/${id}/edit`, {
    method: "GET",
  })) as PurchaseOrdersResponse;
}

export async function createLegacyPurchaseOrder(
  record: LegacyPurchaseOrderLike,
): Promise<PurchaseOrdersResponse> {
  return (await pb.send("/api/purchase_orders/legacy", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(buildLegacyPurchaseOrderPayload(record)),
  })) as PurchaseOrdersResponse;
}

export async function updateLegacyPurchaseOrder(
  id: string,
  record: LegacyPurchaseOrderLike,
): Promise<PurchaseOrdersResponse> {
  return (await pb.send(`/api/purchase_orders/legacy/${id}`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(buildLegacyPurchaseOrderPayload(record)),
  })) as PurchaseOrdersResponse;
}
