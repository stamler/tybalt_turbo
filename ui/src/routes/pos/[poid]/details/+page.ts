import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";
import type {
  PurchaseOrdersAugmentedResponse,
  ExpensesAugmentedResponse,
} from "$lib/pocketbase-types";
import type { SecondApproversResponse } from "$lib/svelte-types";
import { buildPoApproverRequest, fetchPoSecondApprovers } from "$lib/poApprovers";
import { fetchVisiblePO } from "$lib/poVisibility";

export const load: PageLoad = async ({ params }) => {
  try {
    const po = (await fetchVisiblePO(params.poid)) as PurchaseOrdersAugmentedResponse;

    let secondApproverDiagnostics: SecondApproversResponse | null = null;
    try {
      const request = buildPoApproverRequest({
        division: po.division,
        total: po.total,
        kind: po.kind,
        job: po.job,
        type: po.type,
        date: po.date,
        end_date: po.end_date,
        frequency: po.frequency,
      });
      secondApproverDiagnostics = await fetchPoSecondApprovers(request, null);
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
