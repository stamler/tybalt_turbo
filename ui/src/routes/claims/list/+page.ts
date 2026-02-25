import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import type { ClaimListItem } from "$lib/svelte-types";

export const load: PageLoad = async () => {
  try {
    const items = (await pb.send("/api/claims", { method: "GET" })) as ClaimListItem[];
    return { items };
  } catch (error) {
    console.error(`loading claims list: ${error}`);
    return { items: [] as ClaimListItem[] };
  }
};
