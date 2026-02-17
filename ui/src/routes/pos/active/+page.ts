import type { PurchaseOrdersAugmentedResponse } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import { fetchVisiblePOs } from "$lib/poVisibility";
export const load: PageLoad = async () => {
  try {
    const result = (await fetchVisiblePOs("active")) as PurchaseOrdersAugmentedResponse[];
    return {
      items: result,
      realtime_source: "visible" as const,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
  }
};
