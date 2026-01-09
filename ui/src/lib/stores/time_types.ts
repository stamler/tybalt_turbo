import type { TimeTypesResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export const timeTypes = createCollectionStore<TimeTypesResponse>(
  "time_types",
  {
    sort: "code",
    requestKey: "tt",
  },
  {
    fields: ["name", "code", "description"],
    storeFields: ["id", "name", "code", "description"],
  },
  async (item) => {
    // Fetch the new record and add to store
    const fullRecord = await pb.collection("time_types").getOne<TimeTypesResponse>(item.id);
    timeTypes.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    // Fetch the updated record and add to store
    const fullRecord = await pb.collection("time_types").getOne<TimeTypesResponse>(item.id);
    timeTypes.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
);
