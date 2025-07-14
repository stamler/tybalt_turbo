import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

// Custom API response matching /api/clients
export interface Contact {
  id: string;
  given_name: string;
  surname: string;
  email: string;
}

export interface ClientApiResponse {
  id: string;
  name: string;
  contacts: Contact[];
}

const fetchAllClients = async (): Promise<ClientApiResponse[]> =>
  pb.send("/api/clients", { method: "GET" });

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const clients = createCollectionStore<any>(
  "clients",
  {},
  {
    fields: ["id", "name", "contacts"],
    storeFields: ["id", "name", "contacts"],
    extractField: (doc, field) => (doc as Record<string, unknown>)[field] as string,
  },
  // onCreate
  async (item) => {
    const record: ClientApiResponse = await pb.send(`/api/clients/${item.id}`, { method: "GET" });
    clients.update((s) => ({
      ...s,
      items: [...s.items, record],
      index: s.index?.add(record) || s.index,
    }));
  },
  // onUpdate – re-fetch and replace existing entry in the store
  /*
   * NOTE: ClientsEditor.svelte calls clients.refresh(id) after it creates,
   * updates or deletes related client_contacts. That manual refresh invokes
   * this onUpdate callback a second time for the same client (once for the
   * original client.save / contact mutation via realtime, and once for the
   * explicit refresh).  It's small data so we accept the duplication; this
   * callback still ensures the store always has the latest contacts array.
   * We do this because realtime events fire only for the `client_contacts`
   * collection, not for the parent client record; without the manual refresh
   * the contacts list in the UI would lag behind the database.
   */
  async (item) => {
    const fullRecord: ClientApiResponse = await pb.send(`/api/clients/${item.id}`, {
      method: "GET",
    });
    clients.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
  // proxy collection
  "clients",
  fetchAllClients,
  true,
);
