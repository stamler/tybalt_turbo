import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";
import type {
  PurchaseOrdersAugmentedResponse,
  ExpensesAugmentedResponse,
} from "$lib/pocketbase-types";

export const load: PageLoad = async ({ params }) => {
  try {
    const po = await pb
      .collection("purchase_orders_augmented")
      .getOne<PurchaseOrdersAugmentedResponse>(params.poid);

    // fetch related expenses
    const expensesRes: { data: ExpensesAugmentedResponse[] } = await pb.send(
      `/api/expenses/list?purchase_order=${params.poid}`,
      { method: "GET" },
    );
    const expenses = expensesRes?.data ?? [];

    return { po, expenses };
  } catch (err) {
    console.error(`loading purchase order details: ${err}`);
    throw error(404, `Failed to load purchase order details`);
  }
};
