import type { RateRolesResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export const rateRoles = createCollectionStore<RateRolesResponse>(
  "rate_roles",
  {
    sort: "name",
    requestKey: "rate_roles",
  },
  {
    fields: ["name"],
    storeFields: ["id", "name"],
  },
  async (item) => {
    // Fetch the new record and add to store
    const fullRecord = await pb.collection("rate_roles").getOne<RateRolesResponse>(item.id);
    rateRoles.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    // Fetch the updated record and add to store
    const fullRecord = await pb.collection("rate_roles").getOne<RateRolesResponse>(item.id);
    rateRoles.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
);
