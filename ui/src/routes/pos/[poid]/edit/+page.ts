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

export const load: PageLoad<PurchaseOrdersPageData> = async ({ params }) => {
  const defaultItem: Partial<PurchaseOrdersRecord> = {
    po_number: "",
    status: PurchaseOrdersStatusOptions.Unapproved,
    uid: "",
    type: PurchaseOrdersTypeOptions.Normal,
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    end_date: "",
    frequency: PurchaseOrdersFrequencyOptions.Weekly,
    division: "vccd5fo56ctbigh",
    description: "",
    total: 0,
    payment_type: PurchaseOrdersPaymentTypeOptions.OnAccount,
    vendor: "",
    job: "",
    category: "",
    kind: "",
    // approver is configured as not required in pocketbase so we do not have to
    // set it here, but is set by the server side hook
  };

  try {
    // Fetch the purchase order
    const item = await pb.collection("purchase_orders").getOne(params.poid);

    // Fetch approvers using GET query params.
    const queryParams = new URLSearchParams({
      division: item.division,
      amount: String(item.total),
      kind: item.kind || "",
      has_job: String(!!item.job),
      type: item.type === PurchaseOrdersTypeOptions.Recurring ? "Recurring" : item.type,
      start_date: item.date || "",
      end_date: item.end_date || "",
      frequency: item.frequency || "",
    });

    const approvers = await pb.send(`/api/purchase_orders/approvers?${queryParams.toString()}`, {
      method: "GET",
    });

    // Fetch second approvers
    const secondApprovers = await pb.send(
      `/api/purchase_orders/second_approvers?${queryParams.toString()}`,
      {
        method: "GET",
      },
    );

    return {
      item,
      editing: true,
      id: params.poid,
      approvers: approvers,
      second_approvers: secondApprovers,
    };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return {
      item: defaultItem as PurchaseOrdersRecord,
      editing: false,
      id: null,
      approvers: [],
      second_approvers: [],
    };
  }
};
