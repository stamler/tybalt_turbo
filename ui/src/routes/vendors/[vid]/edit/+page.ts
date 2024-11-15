import type { VendorsResponse } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { VendorsPageData } from "$lib/svelte-types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad<VendorsPageData> = async ({ params }) => {
  const defaultItem = {
    name: "",
    alias: "",
    status: "Active",
  } as VendorsResponse;

  try {
    const item = await pb.collection("vendors").getOne<VendorsResponse>(params.vid);
    return { item, editing: true, id: params.vid };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return {
      item: defaultItem,
      editing: false,
      id: null,
    };
  }
};
