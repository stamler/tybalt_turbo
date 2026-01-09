import type { DivisionsResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export const divisions = createCollectionStore<DivisionsResponse>(
  "divisions",
  {
    sort: "code",
    requestKey: "divisions",
  },
  {
    fields: ["code", "name"],
    storeFields: ["id", "code", "name"],
  },
  async (item) => {
    // Fetch the new record and add to store
    const fullRecord = await pb.collection("divisions").getOne<DivisionsResponse>(item.id);
    divisions.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    // Fetch the updated record and add to store
    const fullRecord = await pb.collection("divisions").getOne<DivisionsResponse>(item.id);
    divisions.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
);
