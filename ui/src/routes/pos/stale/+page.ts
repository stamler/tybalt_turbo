import type { PageLoad } from "./$types";
import { fetchVisiblePOs } from "$lib/poVisibility";
export const load: PageLoad = async () => {
  // 30 days ago as YYYY-MM-DD
  const staleDate = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().replace("T", " ");
  try {
    const result = await fetchVisiblePOs("stale", staleDate);
    return {
      items: result,
      realtime_source: "visible" as const,
    };
  } catch (error) {
    console.error(`loading data: ${error}`);
    return { items: [], realtime_source: "visible" as const };
  }
};
