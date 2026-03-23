import type { PageLoad } from "./$types";
import { fetchVisiblePOs } from "$lib/poVisibility";
export const load: PageLoad = async () => {
  try {
    const result = await fetchVisiblePOs("active");
    return {
      items: result,
      realtime_source: "visible" as const,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
    return { items: [], realtime_source: "visible" as const };
  }
};
