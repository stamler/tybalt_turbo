import type {
  ExpensesRecord,
  PurchaseOrdersPaymentTypeOptions,
  PurchaseOrdersStatusOptions,
  PurchaseOrdersTypeOptions,
} from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { ExpensesPageData } from "$lib/svelte-types";
import { pb } from "$lib/pocketbase";

type VisiblePurchaseOrderResponse = {
  id: string;
  po_number: string;
  type: PurchaseOrdersTypeOptions;
  payment_type: PurchaseOrdersPaymentTypeOptions;
  status: PurchaseOrdersStatusOptions;
  division: string;
  description: string;
  vendor: string;
  job: string;
  category: string;
  kind: string;
  recurring_expected_occurrences: number;
  recurring_remaining_occurrences: number;
  cumulative_remaining_balance: number;
};

export const load: PageLoad<ExpensesPageData> = async ({ params }) => {
  const defaultItem = {
    uid: "",
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    division: "",
    description: "",
    payment_type: "OnAccount",
    purchase_order: "",
    vendor: "",
    job: "",
    category: "",
    kind: "",
    // approver is configured as not required in pocketbase so we do not have to
    // set it here, but is set by the server side hook
  };
  let linkedPurchaseOrder: ExpensesPageData["linked_purchase_order"] | undefined = undefined;

  // if the url has a poid param, fetch the purchase_order with that id from the
  // purchase_orders collection in pocketbase and use it to populate the default
  // item
  if (params.poid) {
    const result = (await pb.send(`/api/purchase_orders/visible/${params.poid}`, {
      method: "GET",
    })) as VisiblePurchaseOrderResponse;
    defaultItem.division = result.division;
    defaultItem.description = result.description;
    defaultItem.payment_type = result.payment_type;

    defaultItem.purchase_order = result.id;
    defaultItem.vendor = result.vendor ?? "";
    defaultItem.job = result.job ?? "";
    defaultItem.category = result.category ?? "";
    defaultItem.kind = result.kind ?? "";
    linkedPurchaseOrder = {
      id: result.id,
      po_number: result.po_number,
      type: result.type,
      payment_type: result.payment_type,
      status: result.status,
      recurring_expected_occurrences: result.recurring_expected_occurrences,
      recurring_remaining_occurrences: result.recurring_remaining_occurrences,
      cumulative_remaining_balance: result.cumulative_remaining_balance,
    };
  }

  return {
    item: { ...defaultItem } as ExpensesRecord,
    editing: false,
    id: null,
    linked_purchase_order: linkedPurchaseOrder,
  };
};
