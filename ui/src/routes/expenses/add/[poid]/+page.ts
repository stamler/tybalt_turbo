import type { ExpensesRecord } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { ExpensesPageData } from "$lib/svelte-types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad<ExpensesPageData> = async ({ params }) => {

  const defaultItem = {
    uid: "",
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    division: "vccd5fo56ctbigh",
    description: "",
    payment_type: "OnAccount",
    purchase_order: "",
    vendor_name: "",
    job: "",
    category: "",
    // approver is configured as not required in pocketbase so we do not have to
    // set it here, but is set by the server side hook
  };

  // if the url has a poid param, fetch the purchase_order with that id from the
  // purchase_orders collection in pocketbase and use it to populate the default
  // item
  if (params.poid) {
    console.log("loading", params.poid);
    const result = await pb.collection('purchase_orders').getOne(params.poid);
    defaultItem.division = result.division;
    defaultItem.description = result.description;

    defaultItem.purchase_order = result.id;
    defaultItem.vendor_name = result.vendor_name ?? "";
    defaultItem.job = result.job ?? "";
    defaultItem.category = result.category ?? "";
  }

  return { item: { ...defaultItem } as ExpensesRecord, editing: false, id: null };
};
