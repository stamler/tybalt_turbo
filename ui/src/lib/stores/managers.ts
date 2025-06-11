import type { ManagersResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export const managers = createCollectionStore<ManagersResponse>(
  "managers",
  {
    sort: "surname,given_name",
    requestKey: "managers",
  },
  {
    fields: ["id", "surname", "given_name"],
    storeFields: ["id", "surname", "given_name"],
  },
  async (item) => {
    // Fetch the new record and add to store
    const fullRecord = await pb.collection("managers").getOne<ManagersResponse>(item.id);
    managers.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    // Fetch the updated record and add to store
    const fullRecord = await pb.collection("managers").getOne<ManagersResponse>(item.id);
    managers.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
  "user_claims", // subscribe to user_claims instead of the managers view
);
