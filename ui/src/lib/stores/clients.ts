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
    // This function doesn't capture changes to the save() function in
    // ClientsEditor.svelte creates or updates the client which immediately
    // triggers this function, running it before the client_contacts are
    // subsequently updated in the database. We preserve this update function
    // here nonetheless, because in the case where the client_contacts are not
    // updated in the database, this function will still update the client in
    // the store correctly. However to resolve the issue where the
    // client_contacts are updated in the database, we need to manually reload
    // the client in the clients store. We do this by calling the refresh()
    // function in the clients store and specifying the client id, which
    // actually calls *this* function since it's the onUpdate callback. This
    // means that the onUpdate callback is called twice for the same client,
    // once when the client is created or updated, and once when the
    // client_contacts are updated. While this isn't efficient, it's very little
    // data.
    clients.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
);
