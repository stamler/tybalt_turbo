import type { VendorsResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export const vendors = createCollectionStore<VendorsResponse>(
  "vendors",
  {
    filter: "status = 'Active'",
    requestKey: "vendors",
  },
  {
    fields: ["id", "name", "alias"],
    storeFields: ["id", "name", "alias"],
  },
  async (item) => {
    // Fetch the new record and add to store
    const fullRecord = await pb.collection("vendors").getOne<VendorsResponse>(item.id);
    vendors.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    // Fetch the updated record and add to store
    const fullRecord = await pb.collection("vendors").getOne<VendorsResponse>(item.id);
    vendors.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
);
