import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";
import type { ExpensesAugmentedResponse } from "$lib/pocketbase-types";
import type { PurchaseOrderDetailsPageData, SecondApproversResponse } from "$lib/svelte-types";
import { buildPoApproverRequest, fetchPoSecondApprovers } from "$lib/poApprovers";
import { fetchPendingPO, fetchVisiblePO } from "$lib/poVisibility";

export const load: PageLoad = async ({ params }) => {
  try {
    const po = await fetchVisiblePO(params.poid);
    let canApproveOrReject = false;

    try {
      await fetchPendingPO(params.poid);
      canApproveOrReject = true;
    } catch (pendingErr: any) {
      if (pendingErr?.status !== 404) {
        console.error(`loading pending-approval state: ${pendingErr}`);
      }
    }

    let secondApproverDiagnostics: SecondApproversResponse | null = null;
    try {
      const request = buildPoApproverRequest({
        division: po.division,
        total: po.total,
        currency: po.currency,
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

    // Fetch related expenses through the dedicated PO route.
    //
    // Do not use /api/expenses/list?purchase_order=... here:
    // that endpoint is intentionally "my expenses" scoped and will hide other
    // users' linked expenses, which previously caused PO details pages to show
    // Expenses(0) even when PO aggregates reported committed expenses.
    //
    // The dedicated route still does NOT mean "all expenses on this PO".
    // Backend policy deliberately applies expense-level visibility after PO
    // visibility, so some callers can open a PO details page and still receive
    // only a subset of linked expenses.
    const expensesRes: { data: ExpensesAugmentedResponse[] } = await pb.send(
      `/api/purchase_orders/visible/${params.poid}/expenses`,
      { method: "GET" },
    );
    const expenses = expensesRes?.data ?? [];

    const pageData: PurchaseOrderDetailsPageData = {
      po,
      expenses,
      secondApproverDiagnostics,
      canApproveOrReject,
    };
    return pageData;
  } catch (err) {
    console.error(`loading purchase order details: ${err}`);
    throw error(404, `Failed to load purchase order details`);
  }
};
