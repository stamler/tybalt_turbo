import type { PurchaseOrdersRecord } from "$lib/pocketbase-types";
import {
  PurchaseOrdersFrequencyOptions,
  PurchaseOrdersStatusOptions,
  PurchaseOrdersTypeOptions,
} from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { PurchaseOrdersPageData } from "$lib/svelte-types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";

export const load: PageLoad<PurchaseOrdersPageData> = async ({ params }) => {
  // Get the parent PO
  const parentPo = await pb.collection("purchase_orders").getOne(params.poid);
  if (!parentPo) {
    throw error(404, "Parent PO not found");
  }

  // Fetch approvers using the new API endpoints
  const approvers = await pb.send(`/api/purchase_orders/approvers/${parentPo.division}/0`, {
    method: "GET",
  });

  // Fetch second approvers
  const secondApprovers = await pb.send(
    `/api/purchase_orders/second_approvers/${parentPo.division}/0`,
    {
      method: "GET",
    },
  );

  // Create a new PO with fields that must match the parent
  const defaultItem: Partial<PurchaseOrdersRecord> = {
    po_number: "",
    status: PurchaseOrdersStatusOptions.Unapproved,
    uid: "",
    type: PurchaseOrdersTypeOptions.Normal,
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
    second_approvers: secondApprovers,
    parent_po_number: parentPo.po_number,
  };
};
