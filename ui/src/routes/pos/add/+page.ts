import type { PurchaseOrdersRecord } from "$lib/pocketbase-types";
import {
  PurchaseOrdersStatusOptions,
  PurchaseOrdersTypeOptions,
  PurchaseOrdersFrequencyOptions,
  PurchaseOrdersPaymentTypeOptions,
} from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { PurchaseOrdersPageData } from "$lib/svelte-types";
import { pb } from "$lib/pocketbase";
export const load: PageLoad<PurchaseOrdersPageData> = async () => {
  // Default division and amount for initial load
  const defaultDivision = "vccd5fo56ctbigh";
  const defaultAmount = 0;

  // Fetch approvers using the new API endpoints
  const approvers = await pb.send(
    `/api/purchase_orders/approvers/${defaultDivision}/${defaultAmount}`,
    {
      method: "GET",
    },
  );

  const secondApprovers = await pb.send(
    `/api/purchase_orders/second_approvers/${defaultDivision}/${defaultAmount}`,
    {
      method: "GET",
    },
  );

  const defaultItem: Partial<PurchaseOrdersRecord> = {
    po_number: "",
    status: PurchaseOrdersStatusOptions.Unapproved,
    uid: "",
    type: PurchaseOrdersTypeOptions.Normal,
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    end_date: "",
    frequency: PurchaseOrdersFrequencyOptions.Weekly,
    division: defaultDivision,
    description: "",
    total: 0,
    payment_type: PurchaseOrdersPaymentTypeOptions.OnAccount,
    vendor: "",
    job: "",
    category: "",
    // approver is configured as not required in pocketbase so we do not have to
    // set it here, but is set by the server side hook
  };
  return {
    item: { ...defaultItem } as PurchaseOrdersRecord,
    editing: false,
    id: null,
    approvers: approvers,
    second_approvers: secondApprovers,
  };
};
