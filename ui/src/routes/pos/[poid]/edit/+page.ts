import type { PurchaseOrdersRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { PurchaseOrdersPageData } from "$lib/svelte-types";

export const load: PageLoad<PurchaseOrdersPageData> = async ({ params }) => {
  const defaultItem = {
    po_number: "",
    status: "Unapproved",
    uid: "",
    type: "Normal",
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    end_date: "",
    frequency: "",
    division: "vccd5fo56ctbigh",
    description: "",
    total: 0,
    payment_type: "OnAccount",
    vendor_name: "",
    job: "",
    category: "",
    // approver is configured as not required in pocketbase so we do not have to
    // set it here, but is set by the server side hook
  };
  let item: PurchaseOrdersRecord;
  try {
    item = await pb.collection("purchase_orders").getOne(params.poid);
    return { item, editing: true, id: params.poid };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return { item: { ...defaultItem } as PurchaseOrdersRecord, editing: false, id: null };
  }
};
