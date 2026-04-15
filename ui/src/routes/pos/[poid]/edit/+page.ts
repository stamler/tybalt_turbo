import type { PurchaseOrdersRecord } from "$lib/pocketbase-types";
import {
  PurchaseOrdersStatusOptions,
  PurchaseOrdersTypeOptions,
  PurchaseOrdersFrequencyOptions,
  PurchaseOrdersPaymentTypeOptions,
} from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { PurchaseOrdersPageData } from "$lib/svelte-types";
import { isRedirect, redirect } from "@sveltejs/kit";

export const load: PageLoad<PurchaseOrdersPageData> = async ({ params }) => {
  const defaultItem: Partial<PurchaseOrdersRecord> = {
    po_number: "",
    status: PurchaseOrdersStatusOptions.Unapproved,
    uid: "",
    type: PurchaseOrdersTypeOptions["One-Time"],
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    end_date: "",
    frequency: PurchaseOrdersFrequencyOptions.Weekly,
    division: "vccd5fo56ctbigh",
    description: "",
    total: 0,
    currency: "",
    approval_total_home: 0,
    covered_within_project_budget: false,
    payment_type: PurchaseOrdersPaymentTypeOptions.OnAccount,
    vendor: "",
    job: "",
    category: "",
    kind: "",
    // approver is configured as not required in pocketbase so we do not have to
    // set it here, but is set by the server side hook
  };

  try {
    const item = await pb.collection("purchase_orders").getOne(params.poid);
    if (item.legacy_manual_entry) {
      throw redirect(303, `/pos/legacy/${params.poid}/edit`);
    }
    if (item.status !== PurchaseOrdersStatusOptions.Unapproved) {
      throw redirect(303, `/pos/${params.poid}/details`);
    }

    let parentCurrency = "";
    if (item.parent_po) {
      try {
        const parentPo = await pb.collection("purchase_orders").getOne(item.parent_po);
        parentCurrency = parentPo.currency ?? "";
      } catch (parentError) {
        console.error(`error loading parent PO currency for ${params.poid}: ${parentError}`);
      }
    }

    return { item, editing: true, id: params.poid, parent_currency: parentCurrency };
  } catch (error) {
    if (isRedirect(error)) {
      throw error;
    }
    console.error(`error loading data, returning default item: ${error}`);
    return { item: defaultItem as PurchaseOrdersRecord, editing: false, id: null };
  }
};
