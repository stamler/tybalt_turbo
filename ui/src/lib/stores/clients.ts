import type { ClientsResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export const clients = createCollectionStore<ClientsResponse>(
  "clients",
  {
    requestKey: "client",
    expand: "client_contacts_via_client",
  },
  {
    fields: ["id", "name"],
    // store the expand field so we can access
    // client_contacts_via_client in the search results
    storeFields: ["id", "name", "expand"],
  },
  async (item) => {
    // Fetch the new record with expand options and add to store
    const fullRecord = await pb.collection("clients").getOne<ClientsResponse>(item.id, {
      expand: "client_contacts_via_client",
    });
    clients.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    // Fetch the updated record with expand options and replace in store
    const fullRecord = await pb.collection("clients").getOne<ClientsResponse>(item.id, {
      expand: "client_contacts_via_client",
    });
    clients.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
);
