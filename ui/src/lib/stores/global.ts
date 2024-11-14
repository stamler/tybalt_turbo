/**
 * The global store is used to load data that is used in multiple places in the
 * app.
 */

import type {
  ClientsResponse,
  TimeTypesResponse,
  DivisionsResponse,
  JobsResponse,
  ManagersResponse,
  TimeSheetsResponse,
  VendorsResponse,
} from "$lib/pocketbase-types";
import { writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
import { ClientResponseError } from "pocketbase";
import MiniSearch from "minisearch";
import type { Readable, Invalidator, Subscriber } from "svelte/store";
import { calculateTallies } from "$lib/utilities";
import type { TimeSheetTally } from "$lib/utilities";

interface StoreItem<T> {
  items: T;
  maxAge: number;
  lastRefresh: Date;
}

export type CollectionName =
  | "clients"
  | "time_types"
  | "divisions"
  | "jobs"
  | "vendors"
  | "managers"
  | "time_sheets"
  | "time_sheets_tallies";
type CollectionType = {
  clients: ClientsResponse[];
  time_types: TimeTypesResponse[];
  divisions: DivisionsResponse[];
  jobs: JobsResponse[];
  vendors: VendorsResponse[];
  managers: ManagersResponse[];
  time_sheets: TimeSheetsResponse[];
  time_sheets_tallies: TimeSheetTally[];
};

interface ErrorMessage {
  message: string;
  id: string;
}

interface StoreState {
  collections: {
    [K in CollectionName]: StoreItem<CollectionType[K]>;
  };
  jobsIndex: MiniSearch<JobsResponse> | null;
  clientsIndex: MiniSearch<ClientsResponse> | null;
  vendorsIndex: MiniSearch<VendorsResponse> | null;
  isLoading: boolean;
  error: ClientResponseError | null;
  errorMessages: ErrorMessage[];
}

// Define a type for the wrapped store value
type WrappedStoreValue = {
  [K in CollectionName]: CollectionType[K];
} & Omit<StoreState, "collections"> & {
    collections: StoreState["collections"];
    errorMessages: ErrorMessage[];
    addError: (message: string) => void;
    dismissError: (id: string) => void;
  };

const createStore = () => {
  const { subscribe, update } = writable<StoreState>({
    collections: {
      // 1 day
      time_types: { items: [], maxAge: 86400 * 1000, lastRefresh: new Date(0) },
      divisions: { items: [], maxAge: 86400 * 1000, lastRefresh: new Date(0) },
      time_sheets: { items: [], maxAge: 86400 * 1000, lastRefresh: new Date(0) },
      time_sheets_tallies: { items: [], maxAge: 86400 * 1000, lastRefresh: new Date(0) },
      // 5 minutes
      jobs: { items: [], maxAge: 5 * 60 * 1000, lastRefresh: new Date(0) },
      // 1 hour
      vendors: { items: [], maxAge: 3600 * 1000, lastRefresh: new Date(0) },
      managers: { items: [], maxAge: 3600 * 1000, lastRefresh: new Date(0) },
      clients: { items: [], maxAge: 3600 * 1000, lastRefresh: new Date(0) },
    },
    jobsIndex: null,
    clientsIndex: null,
    vendorsIndex: null,
    isLoading: false,
    error: null,
    errorMessages: [],
  });

  const loadData = async <K extends CollectionName>(key: K) => {
    update((state) => ({ ...state, isLoading: true, error: null }));
    try {
      let items: CollectionType[typeof key];
      const userId = get(authStore)?.model?.id || "";
      switch (key) {
        case "clients":
          items = (await pb.collection("clients").getFullList<ClientsResponse>({
            requestKey: "client",
            expand: "contacts_via_client",
          })) as CollectionType[typeof key];
          break;
        case "time_types":
          items = (await pb.collection("time_types").getFullList<TimeTypesResponse>({
            sort: "code",
            requestKey: "tt",
          })) as CollectionType[typeof key];
          break;
        case "divisions":
          items = (await pb.collection("divisions").getFullList<DivisionsResponse>({
            sort: "code",
            requestKey: "div",
          })) as CollectionType[typeof key];
          break;
        case "jobs":
          items = (await pb.collection("jobs").getFullList<JobsResponse>({
            expand: "categories_via_job,client",
            sort: "-number",
            requestKey: "job",
          })) as CollectionType[typeof key];
          break;
        case "managers":
          items = (await pb.collection("managers").getFullList<ManagersResponse>({
            requestKey: "manager",
          })) as CollectionType[typeof key];
          break;
        case "vendors":
          items = (await pb.collection("vendors").getFullList<VendorsResponse>({
            requestKey: "vendor",
          })) as CollectionType[typeof key];
          break;
        case "time_sheets":
          items = (await pb.collection("time_sheets").getFullList<TimeSheetsResponse>({
            requestKey: "time_sheets",
            filter: pb.filter("uid={:userId}", { userId }),
            expand:
              "time_entries_via_tsid.time_type,time_entries_via_tsid.job,time_entries_via_tsid.division",
            sort: "-week_ending",
          })) as CollectionType[typeof key];
          break;
      }

      update((state) => {
        const newState = { ...state };
        newState.collections[key] = {
          ...newState.collections[key],
          items,
          lastRefresh: new Date(),
        };

        if (key === "jobs") {
          const jobsIndex = new MiniSearch<JobsResponse>({
            fields: ["id", "number", "description", "expand.client.name"],
            storeFields: ["id", "number", "description", "expand.client.name"],
          });
          jobsIndex.addAll(items as JobsResponse[]);
          newState.jobsIndex = jobsIndex;
        }

        if (key === "vendors") {
          const vendorsIndex = new MiniSearch<VendorsResponse>({
            fields: ["id", "name", "alias"],
            storeFields: ["id", "name", "alias"],
          });
          vendorsIndex.addAll(items as VendorsResponse[]);
          newState.vendorsIndex = vendorsIndex;
        }

        if (key === "clients") {
          const clientsIndex = new MiniSearch<ClientsResponse>({
            fields: ["id", "name"],
            storeFields: ["id", "name"],
          });
          clientsIndex.addAll(items as ClientsResponse[]);
          newState.clientsIndex = clientsIndex;
        }

        if (key === "time_sheets") {
          const tallies = items.map((item) => calculateTallies(item as TimeSheetsResponse));
          newState.collections.time_sheets_tallies = {
            items: tallies,
            lastRefresh: new Date(),
            maxAge: newState.collections.time_sheets.maxAge,
          };
        }

        return { ...newState, isLoading: false, error: null };
      });
    } catch (error: unknown) {
      const typedErr = error as ClientResponseError;
      console.error(`Error loading ${key}:`, typedErr);
      update((state) => ({ ...state, isLoading: false, error: typedErr }));
    }
  };

  const refresh = async (key: CollectionName | null = null) => {
    // refresh() should no-op if the user is not logged in. Failure to do so
    // will cause the lastRefresh date to be set to now, which will prevent
    // subsequent refreshes from happening until maxAge milliseconds have passed
    // leaving blank data on the UI because calling the backend with no auth
    // token will return no results.
    if (!get(authStore)?.isValid) {
      console.log("User is not logged in, skipping refresh");
      return;
    }

    update((state) => {
      const now = new Date();
      const newState = { ...state };

      if (key !== null) {
        // refresh immediately when the key is specified
        loadData(key);
      } else {
        // if the key is not specified, refresh all collections that are older
        // than their maxAge
        for (const k of Object.keys(newState.collections) as CollectionName[]) {
          const item = newState.collections[k];
          if (now.getTime() - item.lastRefresh.getTime() >= item.maxAge) {
            loadData(k);
          }
        }
      }

      return newState;
    });
  };

  const addError = (message: string) => {
    update((state) => {
      const id = crypto.randomUUID();
      return {
        ...state,
        errorMessages: [...state.errorMessages, { message, id }],
      };
    });
  };

  const dismissError = (id: string) => {
    update((state) => ({
      ...state,
      errorMessages: state.errorMessages.filter((error) => error.id !== id),
    }));
  };

  const deleteItem = async <K extends CollectionName>(collectionName: K, id: string) => {
    try {
      await pb.collection(collectionName).delete(id);

      // TODO: This is quite inefficient because it reloads the entire
      // collection, however the collection could change to it makes sense to
      // just reload the data somewhat frequently. Perhaps we could just delete
      // the item from the list and keep the rest of the collection state around
      // to avoid reloading the entire collection, but this will involve a
      // different function to handle the fact that some collections also have
      // MiniSearch indexes.
      refresh(collectionName);
    } catch (error) {
      addError(`error deleting item: ${error}`);
    }
  };

  return {
    subscribe,
    refresh,
    addError,
    dismissError,
    deleteItem,
  };
};

const _globalStore = createStore();

// Proxy handler to allow access like $globalStore.time_types
const proxyHandler: ProxyHandler<StoreState> = {
  get(target, prop: string | symbol) {
    if (prop in target.collections) {
      const items = target.collections[prop as CollectionName].items;
      return items || []; // Return an empty array if items is undefined
    }
    return target[prop as keyof StoreState];
  },
};

// Wrapped store that provides access to the collections directly
const wrappedStore: Readable<WrappedStoreValue> & {
  refresh: typeof _globalStore.refresh;
  addError: typeof _globalStore.addError;
  dismissError: typeof _globalStore.dismissError;
  deleteItem: typeof _globalStore.deleteItem;
} = {
  subscribe: (run: Subscriber<WrappedStoreValue>, invalidate?: Invalidator<WrappedStoreValue>) => {
    return _globalStore.subscribe(
      (value) => run(new Proxy(value, proxyHandler) as unknown as WrappedStoreValue),
      invalidate as Invalidator<StoreState> | undefined,
    );
  },
  refresh: _globalStore.refresh,
  addError: _globalStore.addError,
  dismissError: _globalStore.dismissError,
  deleteItem: _globalStore.deleteItem,
};

export const globalStore = wrappedStore;
