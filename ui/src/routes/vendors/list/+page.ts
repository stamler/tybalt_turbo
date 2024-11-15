import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import type { VendorsResponse } from "$lib/pocketbase-types";

export const load: PageLoad = async () => {
  try {
    const items = await pb.collection("vendors").getFullList<VendorsResponse>();
    return { items };
  } catch (error) {
    console.error(`error loading vendors: ${error}`);
    return { items: [] };
  }
};
