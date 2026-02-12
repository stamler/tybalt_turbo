import type { PurchaseOrdersRecord } from "$lib/pocketbase-types";
import {
  PurchaseOrdersFrequencyOptions,
  PurchaseOrdersStatusOptions,
  PurchaseOrdersTypeOptions,
} from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { PurchaseOrdersPageData, SecondApproversResponse } from "$lib/svelte-types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";

export const load: PageLoad<PurchaseOrdersPageData> = async ({ params }) => {
  // Get the parent PO
  const parentPo = await pb.collection("purchase_orders").getOne(params.poid);
  if (!parentPo) {
    throw error(404, "Parent PO not found");
  }

  // Fetch approvers using GET query params.
  const queryParams = new URLSearchParams({
    division: parentPo.division,
    amount: "0",
    kind: parentPo.kind || "",
    has_job: String(!!parentPo.job),
    type: "One-Time",
    start_date: parentPo.date || "",
    end_date: "",
    frequency: "",
  });

  const approvers = await pb.send(`/api/purchase_orders/approvers?${queryParams.toString()}`, {
    method: "GET",
  });

  // Fetch second approvers
  const secondApproversResponse = (await pb.send(
    `/api/purchase_orders/second_approvers?${queryParams.toString()}`,
    {
      method: "GET",
    },
  )) as SecondApproversResponse;

  // Create a new PO with fields that must match the parent
  const defaultItem: Partial<PurchaseOrdersRecord> = {
    po_number: "",
    status: PurchaseOrdersStatusOptions.Unapproved,
    uid: "",
    type: PurchaseOrdersTypeOptions["One-Time"],
    parent_po: params.poid,
    date: new Date().toISOString().split("T")[0],
    end_date: "",
    frequency: PurchaseOrdersFrequencyOptions.Weekly,
    division: parentPo.division,
    description: parentPo.description,
    total: 0,
    payment_type: parentPo.payment_type,
    vendor: parentPo.vendor,
    job: parentPo.job,
    category: parentPo.category,
    kind: parentPo.kind || "",
    approver: "",
    attachment: "",
  };

  return {
    // we cast here rather than using defaultItem directly because some fields
    // from PurchaseOrdersRecord are not present in the defaultItem and if they
    // were we would get an error on the backend due to isset field restrictions
    // on create.
    item: defaultItem as PurchaseOrdersRecord,
    editing: false,
    id: null,
    approvers: approvers,
    second_approvers: secondApproversResponse.approvers,
    parent_po_number: parentPo.po_number,
  };
};
