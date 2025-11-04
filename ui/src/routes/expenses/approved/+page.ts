import type { ExpensesAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
export const load: PageLoad = async () => {
  try {
    const result: { data: ExpensesAugmentedResponse[]; total_pages?: number; limit?: number } = await pb.send(`/api/expenses/approved`, {
      method: "GET",
    });
    return {
      items: result?.data ?? [],
      totalPages: result?.total_pages ?? 0,
      limit: result?.limit ?? 20,
      // createdItemIsVisible: (record: ExpensesResponse) => {
      //   return record.approved !== "" && record.approver === userId;
      // },
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
