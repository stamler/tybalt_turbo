import type { VendorsRecord } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { VendorsPageData } from "$lib/svelte-types";

export const load: PageLoad<VendorsPageData> = async () => {
  const defaultItem = {
    name: "",
    alias: "",
    status: "Active",
  } as VendorsRecord;

  return {
    item: defaultItem,
    editing: false,
    id: null,
  };
};
