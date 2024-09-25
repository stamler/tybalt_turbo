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
    total: 100, // total cannot be 0 due to schema constraint. This number will be
    // overridden by the backend to the actual total if the payment_type is
    // Mileage or Allowance because these types are calculated based on current
    // rates. If the payment_type is not Mileage or Allowance, the total will be
    // used as is.
    payment_type: "OnAccount",
    vendor_name: "",
    job: "",
    category: "",
    // approver is configured as not required in pocketbase so we do not have to
    // set it here, but is set by the server side hook
  };
  let item: ExpensesRecord;
  try {
    item = await pb.collection("expenses").getOne(params.eid);
    return { item, editing: true, id: params.eid };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return { item: { ...defaultItem } as ExpensesRecord, editing: false, id: null };
  }
};
