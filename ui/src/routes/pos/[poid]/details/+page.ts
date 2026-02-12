import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";
import type {
  PurchaseOrdersAugmentedResponse,
  ExpensesAugmentedResponse,
} from "$lib/pocketbase-types";
import type { SecondApproversResponse } from "$lib/svelte-types";

export const load: PageLoad = async ({ params }) => {
  try {
    const po = await pb
      .collection("purchase_orders_augmented")
      .getOne<PurchaseOrdersAugmentedResponse>(params.poid);

    let secondApproverDiagnostics: SecondApproversResponse | null = null;
    try {
      const queryParams = new URLSearchParams({
        division: po.division || "",
        amount: String(po.total ?? 0),
        kind: po.kind || "",
        has_job: String(!!po.job),
        type: po.type === "Recurring" ? "Recurring" : po.type || "",
        start_date: po.date || "",
        end_date: po.end_date || "",
        frequency: po.frequency || "",
      });
      secondApproverDiagnostics = (await pb.send(
        `/api/purchase_orders/second_approvers?${queryParams.toString()}`,
        { method: "GET", requestKey: null },
      )) as SecondApproversResponse;
    } catch (diagErr) {
      // Keep details view available even if diagnostics fetch fails.
      console.error(`loading second approver diagnostics: ${diagErr}`);
    }

    // fetch related expenses
    const expensesRes: { data: ExpensesAugmentedResponse[] } = await pb.send(
      `/api/expenses/list?purchase_order=${params.poid}`,
      { method: "GET" },
    );
    const expenses = expensesRes?.data ?? [];

    return { po, expenses, secondApproverDiagnostics };
  } catch (err) {
    console.error(`loading purchase order details: ${err}`);
    throw error(404, `Failed to load purchase order details`);
  }
};
