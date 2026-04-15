import type { PurchaseOrdersRecord } from "$lib/pocketbase-types";
import {
  PurchaseOrdersFrequencyOptions,
  PurchaseOrdersPaymentTypeOptions,
  PurchaseOrdersStatusOptions,
  PurchaseOrdersTypeOptions,
} from "$lib/pocketbase-types";
import { fetchLegacyPurchaseOrderForEdit } from "$lib/legacyPurchaseOrders";
import type { PageLoad } from "./$types";
import type { PurchaseOrdersPageData } from "$lib/svelte-types";
import { redirect } from "@sveltejs/kit";

const defaultItem = (): PurchaseOrdersRecord =>
  ({
    _imported: false,
    approval_total: 0,
    approved: "",
    approver: "",
    attachment: "",
    branch: "",
    cancelled: "",
    canceller: "",
    category: "",
    closed: "",
    closed_by_system: false,
    covered_within_project_budget: false,
    closer: "",
    created: "",
    date: new Date().toISOString().split("T")[0],
    description: "",
    division: "",
    end_date: "",
    frequency: PurchaseOrdersFrequencyOptions.Weekly,
    id: "",
    job: "",
    kind: "",
    legacy_manual_entry: true,
    parent_po: "",
    payment_type: PurchaseOrdersPaymentTypeOptions.OnAccount,
    po_number: "",
    priority_second_approver: "",
    rejected: "",
    rejection_reason: "",
    rejector: "",
    second_approval: "",
    second_approver: "",
    status: PurchaseOrdersStatusOptions.Active,
    total: 0,
    type: PurchaseOrdersTypeOptions["One-Time"],
    uid: "",
    updated: "",
    vendor: "",
  }) as PurchaseOrdersRecord;

export const load: PageLoad<PurchaseOrdersPageData> = async ({ params }) => {
  try {
    const item = await fetchLegacyPurchaseOrderForEdit(params.poid);
    return { item, editing: true, id: params.poid };
  } catch (error) {
    const responseData = (error as any)?.response?.data;
    const errorCode =
      responseData?.global?.code ?? responseData?.status?.code ?? responseData?.uid?.code ?? "";
    if (errorCode === "legacy_purchase_order_only") {
      throw redirect(303, `/pos/${params.poid}/edit`);
    }
    console.error(`error loading legacy PO ${params.poid}: ${error}`);
    return {
      item: defaultItem(),
      editing: true,
      id: params.poid,
      loadError:
        "This legacy purchase order could not be loaded. It may not exist or you may not have access.",
    };
  }
};
