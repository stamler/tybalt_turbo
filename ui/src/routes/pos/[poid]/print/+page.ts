import type { PageLoad } from "./$types";
import { error, isHttpError } from "@sveltejs/kit";
import { fetchVisiblePO } from "$lib/poVisibility";
import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";

export const load: PageLoad = async ({ params }) => {
  try {
    const po = (await fetchVisiblePO(params.poid)) as PurchaseOrdersAugmentedResponse;

    if (po.status !== "Active") {
      throw error(404, "Printable purchase order not found");
    }

    return { po };
  } catch (err: unknown) {
    if (isHttpError(err)) throw err;

    console.error(`loading printable purchase order: ${err}`);
    throw error(500, "Failed to load printable purchase order");
  }
};
