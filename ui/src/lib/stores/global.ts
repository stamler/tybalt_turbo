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
  VendorsResponse,
  UserPoPermissionDataResponse,
} from "$lib/pocketbase-types";
import { writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
import { ClientResponseError } from "pocketbase";
import MiniSearch from "minisearch";
import type { Readable, Subscriber } from "svelte/store";

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
type CollectionType = {
  clients: ClientsResponse[];
  time_types: TimeTypesResponse[];
  divisions: DivisionsResponse[];
  jobs: JobsResponse[];
  vendors: VendorsResponse[];
  managers: ManagersResponse[];
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
  user_po_permission_data: {
    id: string;
    max_amount: number;
    lower_threshold: number;
    upper_threshold: number;
    divisions: string[];
    claims: string[];
    maxAge: number;
    lastRefresh: Date;
  };
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
    claims: string[];
    user_po_permission_data: {
      id: string;
      max_amount: number;
      lower_threshold: number;
      upper_threshold: number;
      divisions: string[];
      claims: string[];
    };
  };

const createStore = () => {
  const { subscribe, update } = writable<StoreState>({
    collections: {
      // 1 day
      time_types: { items: [], maxAge: 86400 * 1000, lastRefresh: new Date(0) },
      divisions: { items: [], maxAge: 86400 * 1000, lastRefresh: new Date(0) },
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
    user_po_permission_data: {
      id: "",
      max_amount: 0,
      lower_threshold: 0,
      upper_threshold: 0,
      divisions: [],
      claims: [],
      maxAge: 3600 * 1000,
      lastRefresh: new Date(0),
    },
    error: null,
    errorMessages: [],
  });

  const loadUserPoPermissionData = async () => {
    try {
      const userId = get(authStore)?.model?.id || "";
      const userPoPermissionData = await pb
        .collection("user_po_permission_data")
        .getFullList<UserPoPermissionDataResponse>({
          filter: pb.filter("id={:userId}", { userId }),
        });
      update((state) => ({
        ...state,
        user_po_permission_data: {
          // If the user has no user_po_permission_data, set the default values
          ...(userPoPermissionData.length > 0
            ? userPoPermissionData[0]
            : {
                id: "",
                max_amount: 0,
                lower_threshold: 0,
                upper_threshold: 0,
                divisions: [],
                claims: [],
              }),
          lastRefresh: new Date(),
          maxAge: state.user_po_permission_data.maxAge,
        },
      }));
    } catch (error: unknown) {
      const typedErr = error as ClientResponseError;
      console.error("Error loading user po permission data:", typedErr);
    }
  };

  const loadData = async <K extends CollectionName>(key: K) => {
    update((state) => ({ ...state, isLoading: true, error: null }));
    try {
      let items: CollectionType[typeof key];
      switch (key) {
        case "clients":
          items = (await pb.collection("clients").getFullList<ClientsResponse>({
            requestKey: "client",
            expand: "client_contacts_via_client",
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
            fields: ["id", "number", "description", "client"],
            storeFields: ["id", "number", "description", "client"],
            extractField: (document, fieldName) => {
              if (fieldName === "client") {
                return document.expand?.client?.name;
              }
              return document[fieldName as keyof JobsResponse] as string;
            },
          });
          jobsIndex.addAll(items as JobsResponse[]);
          newState.jobsIndex = jobsIndex;
        }

        if (key === "vendors") {
          const vendorsIndex = new MiniSearch<VendorsResponse>({
            fields: ["id", "name", "alias"],
            storeFields: ["id", "name", "alias"],
          });
          // only index Active vendors so users don't see inactive vendors in
          // the vendor dropdown
          vendorsIndex.addAll(
            (items as VendorsResponse[]).filter((item) => item.status === "Active"),
          );
          newState.vendorsIndex = vendorsIndex;
        }

        if (key === "clients") {
          const clientsIndex = new MiniSearch<ClientsResponse>({
            fields: ["id", "name"],
            // store the expand field so we can access
            // client_contacts_via_client in the search results
            storeFields: ["id", "name", "expand"],
          });
          clientsIndex.addAll(items as ClientsResponse[]);
          newState.clientsIndex = clientsIndex;
        }

        return { ...newState, isLoading: false, error: null };
      });
    } catch (error: unknown) {
      const typedErr = error as ClientResponseError;
      console.error(`Error loading ${key}:`, typedErr);
      update((state) => ({ ...state, isLoading: false, error: typedErr }));
    }
  };

  // TODO: instead of manually calling refresh(), we should use the subscribe
  // function to refresh the store based on events.
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

      if (
        now.getTime() - state.user_po_permission_data.lastRefresh.getTime() >=
        state.user_po_permission_data.maxAge
      ) {
        loadUserPoPermissionData();
      }

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
      // MiniSearch indexes. We should probably attempt to use the realtime
      // events to update the collection state instead of reloading the entire
      // collection for the collections that change frequently (such as jobs).
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
    if (prop === "claims") {
      return target.user_po_permission_data.claims;
    }
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
  subscribe: (run: Subscriber<WrappedStoreValue>, invalidate?: () => void) => {
    return _globalStore.subscribe(
      (value) => run(new Proxy(value, proxyHandler) as unknown as WrappedStoreValue),
      invalidate,
    );
  },
  refresh: _globalStore.refresh,
  addError: _globalStore.addError,
  dismissError: _globalStore.dismissError,
  deleteItem: _globalStore.deleteItem,
};

export const globalStore = wrappedStore;
