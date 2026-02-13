// Shared branches collection store.
//
// Why this exists:
// Several editors previously loaded branches with ad-hoc `getFullList` calls.
// Centralizing through `createCollectionStore` gives one source of truth for
// initial load, realtime updates, and search index behavior.
import type { BranchesResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export const branches = createCollectionStore<BranchesResponse>(
  "branches",
  {
    sort: "name",
    requestKey: "branches",
  },
  {
    fields: ["name", "code"],
    storeFields: ["id", "name", "code"],
  },
  async (item) => {
    const fullRecord = await pb.collection("branches").getOne<BranchesResponse>(item.id);
    branches.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    const fullRecord = await pb.collection("branches").getOne<BranchesResponse>(item.id);
    branches.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
);
