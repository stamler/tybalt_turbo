import type { ExpensesResponse } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { ExpensesPageData } from "$lib/svelte-types";
import { pb } from "$lib/pocketbase";

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
    // approver is configured as not required in pocketbase so we do not have to
    // set it here, but is set by the server side hook
  };
  let item: ExpensesResponse;
  try {
    item = await pb.collection("expenses").getOne(params.eid, {
      expand: "purchase_order",
    });
    return { item, editing: true, id: params.eid };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return { item: { ...defaultItem } as ExpensesResponse, editing: false, id: null };
  }
};
