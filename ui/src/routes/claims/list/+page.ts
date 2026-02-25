import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import type { ClaimsResponse } from "$lib/pocketbase-types";

export const load: PageLoad = async () => {
  try {
    const items = await pb
      .collection("claims")
      .getFullList<ClaimsResponse>({ sort: "name" });
    return { items };
  } catch (error) {
    console.error(`loading claims list: ${error}`);
    return { items: [] as ClaimsResponse[] };
  }
};
