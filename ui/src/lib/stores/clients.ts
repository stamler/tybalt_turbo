import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";
import { errorStatus } from "$lib/utilities";
import { get } from "svelte/store";
import type { RecordSubscription, RecordModel, UnsubscribeFunc } from "pocketbase";

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
  alias: string;
  contacts: Contact[];
  referencing_jobs_count: number;
  outstanding_balance: number;
  outstanding_balance_date: string;
  // true when any job, job request, invoicing profile or note uses the client,
  // so the server refuses to delete it
  referenced: boolean;
}

const fetchAllClients = async (): Promise<ClientApiResponse[]> =>
  pb.send("/api/clients", { method: "GET" });

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const store = createCollectionStore<any>(
  "clients",
  {},
  {
    fields: ["name", "alias", "contacts"],
    storeFields: [
      "id",
      "name",
      "alias",
      "contacts",
      "referencing_jobs_count",
      "outstanding_balance",
      "outstanding_balance_date",
      "referenced",
    ],
    extractField: (doc, field) => (doc as Record<string, unknown>)[field] as string,
  },
  // onCreate
  async (item) => {
    const record: ClientApiResponse = await pb.send(`/api/clients/${item.id}`, { method: "GET" });
    store.update((s) => ({
      ...s,
      items: [...s.items, record],
      index: s.index?.add(record) || s.index,
    }));
  },
  // onUpdate – re-fetch and replace existing entry in the store
  /*
   * NOTE: ContactEditor.svelte calls clients.refresh(id) after it creates,
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
    await reloadClient(item.id);
  },
  // proxy collection
  "clients",
  fetchAllClients,
  true,
);

// Re-read one client and replace it in the store. Returns the fresh client, or
// null when it no longer exists, in which case it is removed from the store.
async function reloadClient(id: string): Promise<ClientApiResponse | null> {
  let record: ClientApiResponse;
  try {
    // requestKey null: reloads of one client may overlap, and PocketBase would
    // otherwise cancel the earlier one.
    record = await pb.send(`/api/clients/${id}`, { method: "GET", requestKey: null });
  } catch (error: unknown) {
    if (errorStatus(error) === 404) {
      store.remove(id);
      return null;
    }
    throw error;
  }
  store.update((state) => {
    const present = state.index?.has(id) ?? false;
    if (present) state.index?.replace(record);
    else state.index?.add(record);
    return {
      ...state,
      items: present
        ? state.items.map((i) => (i.id === id ? record : i))
        : [...state.items, record],
    };
  });
  return record;
}

// Realtime events fire only for the client record itself, so watch the
// collections that use clients to keep each client's referenced flag current.
const USE_COLLECTIONS = ["jobs", "job_requests", "client_invoicing_information", "client_notes"];
let unsubscribeUses: Promise<UnsubscribeFunc[]> | null = null;

function onUseEvent(e: RecordSubscription<RecordModel>) {
  const items: ClientApiResponse[] = get(store).items;
  const ids = new Set<string>();
  for (const field of ["client", "job_owner"]) {
    if (e.record[field]) ids.add(e.record[field]);
  }
  for (const id of ids) {
    const item = items.find((i) => i.id === id);
    // Reload only when the event can change the flag: a new use of a client
    // that had none, or a removed use of one that had some. An update event
    // does not carry the record's previous client, so a client a record stops
    // using stays referenced until the list reloads; it keeps Delete hidden,
    // which is the safe direction.
    if (item && item.referenced === (e.action === "delete")) {
      void reloadClient(id).catch(() => {});
    }
  }
}

export const clients = {
  ...store,
  reloadClient,
  init: async () => {
    await store.init();
    if (unsubscribeUses === null && get(store).initialized) {
      // A collection the user cannot subscribe to only leaves its uses stale;
      // the list page still reports a refused delete.
      unsubscribeUses = Promise.all(
        USE_COLLECTIONS.map((c) =>
          pb
            .collection(c)
            .subscribe("*", onUseEvent)
            .catch((): UnsubscribeFunc => async () => {}),
        ),
      );
    }
  },
  unsubscribe: () => {
    void unsubscribeUses?.then((fns) => fns.forEach((fn) => fn()));
    unsubscribeUses = null;
    store.unsubscribe();
  },
};
