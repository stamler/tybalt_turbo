// Shared PO approver API helpers.
//
// Why this file exists:
// The PO editor and PO route loaders were each building the same request params and
// manually calling `/api/purchase_orders/approvers` and `/api/purchase_orders/second_approvers`.
// Keeping that logic in several places increased drift risk whenever the backend
// request contract changed (new fields, normalization rules, request options).
//
// What this file centralizes:
// - The canonical request shape (`PoApproverRequest`).
// - Request normalization from editor/record models (`buildPoApproverRequest`),
//   including amount coercion and `has_job` derivation.
// - Endpoint calls for first approvers and second-approver diagnostics.
// - A bundled helper that fetches both endpoints together for editor workflows.
//
// Notes for maintainers:
// - `requestKey` defaults to null-friendly signatures because some callers opt out
//   of PocketBase auto-cancel while users type quickly.
// - If backend query semantics change, update this file first and keep callers thin.
import { pb } from "$lib/pocketbase";
import type { PoApproversResponse } from "$lib/pocketbase-types";
import type { SecondApproversResponse } from "$lib/svelte-types";

export type PoApproverRequest = {
  division: string;
  amount: string;
  kind: string;
  has_job: string;
  type: string;
  start_date: string;
  end_date: string;
  frequency: string;
};

export function buildPoApproverRequest(params: {
  division?: string;
  total?: number | string;
  kind?: string;
  job?: string;
  type?: string;
  date?: string;
  end_date?: string;
  frequency?: string;
}): PoApproverRequest {
  return {
    division: params.division ?? "",
    amount: String(Number(params.total ?? 0)),
    kind: params.kind ?? "",
    has_job: String((params.job ?? "") !== ""),
    type: params.type ?? "",
    start_date: params.date ?? "",
    end_date: params.end_date ?? "",
    frequency: params.frequency ?? "",
  };
}

function toQueryString(request: PoApproverRequest): string {
  return new URLSearchParams(request).toString();
}

export async function fetchPoApprovers(
  request: PoApproverRequest,
  requestKey: string | null = null,
): Promise<PoApproversResponse[]> {
  return pb.send(`/api/purchase_orders/approvers?${toQueryString(request)}`, {
    method: "GET",
    requestKey,
  });
}

export async function fetchPoSecondApprovers(
  request: PoApproverRequest,
  requestKey: string | null = null,
): Promise<SecondApproversResponse> {
  return pb.send(`/api/purchase_orders/second_approvers?${toQueryString(request)}`, {
    method: "GET",
    requestKey,
  }) as Promise<SecondApproversResponse>;
}

export async function fetchPoApproversBundle(
  request: PoApproverRequest,
  requestKey: string | null = null,
): Promise<{ approvers: PoApproversResponse[]; secondApproversResponse: SecondApproversResponse }> {
  const [approvers, secondApproversResponse] = await Promise.all([
    fetchPoApprovers(request, requestKey),
    fetchPoSecondApprovers(request, requestKey),
  ]);
  return { approvers, secondApproversResponse };
}
