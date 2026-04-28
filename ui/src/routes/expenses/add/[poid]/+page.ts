import type { ExpensesRecord } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { ExpensesPageData } from "$lib/svelte-types";
import { fetchVisiblePO, type VisiblePurchaseOrderResponse } from "$lib/poVisibility";

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
    const result: VisiblePurchaseOrderResponse = await fetchVisiblePO(params.poid);
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
      uid: result.uid,
      po_number: result.po_number,
      currency: result.currency ?? "",
      type: result.type,
      payment_type: result.payment_type,
      status: result.status,
      recurring_expected_occurrences: result.recurring_expected_occurrences,
      recurring_remaining_occurrences: result.recurring_remaining_occurrences,
      remaining_amount: result.remaining_amount,
    };
  }

  return {
    item: { ...defaultItem } as ExpensesRecord,
    editing: false,
    id: null,
    linked_purchase_order: linkedPurchaseOrder,
  };
};
