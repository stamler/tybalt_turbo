/**
 * The global store is used to load data that is used in multiple places in the
 * app.
 */

import type {
  TimeTypesRecord,
  DivisionsRecord,
  JobsRecord,
  ManagersRecord,
} from "$lib/pocketbase-types";
import { writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { ClientResponseError } from "pocketbase";
import MiniSearch from "minisearch";
import type { Readable, Invalidator, Subscriber } from "svelte/store";

interface StoreItem<T> {
  items: T;
  maxAge: number;
  lastRefresh: Date;
}

export type CollectionName = "time_types" | "divisions" | "jobs" | "managers";
type CollectionType = {
  time_types: TimeTypesRecord[];
  divisions: DivisionsRecord[];
  jobs: JobsRecord[];
  managers: ManagersRecord[];
};

interface StoreState {
  collections: {
    [K in CollectionName]: StoreItem<CollectionType[K]>;
  };
  jobsIndex: MiniSearch<JobsRecord> | null;
  isLoading: boolean;
  error: ClientResponseError | null;
}

// Define a type for the wrapped store value
type WrappedStoreValue = {
  [K in CollectionName]: CollectionType[K];
} & Omit<StoreState, "collections"> & {
    collections: StoreState["collections"];
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
      managers: { items: [], maxAge: 3600 * 1000, lastRefresh: new Date(0) },
    },
    jobsIndex: null,
    isLoading: false,
    error: null,
  });

  const loadData = async <K extends CollectionName>(key: K) => {
    update((state) => ({ ...state, isLoading: true, error: null }));
    try {
      let items: CollectionType[typeof key];
      switch (key) {
        case "time_types":
          items = (await pb.collection("time_types").getFullList<TimeTypesRecord>({
            sort: "code",
            requestKey: "tt",
          })) as CollectionType[typeof key];
          break;
        case "divisions":
          items = (await pb.collection("divisions").getFullList<DivisionsRecord>({
            sort: "code",
            requestKey: "div",
          })) as CollectionType[typeof key];
          break;
        case "jobs":
          items = (await pb.collection("jobs").getFullList<JobsRecord>({
            sort: "-number",
            requestKey: "job",
          })) as CollectionType[typeof key];
          break;
        case "managers":
          items = (await pb
            .collection("managers")
            .getFullList<ManagersRecord>({ requestKey: "manager" })) as CollectionType[typeof key];
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
          const jobsIndex = new MiniSearch<JobsRecord>({
            fields: ["id", "number", "description"],
            storeFields: ["id", "number", "description"],
          });
          jobsIndex.addAll(items as JobsRecord[]);
          newState.jobsIndex = jobsIndex;
        }

        return { ...newState, isLoading: false, error: null };
      });
    } catch (error: unknown) {
      const typedErr = error as ClientResponseError;
      console.error(`Error loading ${key}:`, typedErr);
      update((state) => ({ ...state, isLoading: false, error: typedErr }));
    }
  };

  const refresh = async (key: CollectionName = "" as CollectionName) => {
    update((state) => {
      const now = new Date();
      const newState = { ...state };

      if (key) {
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

  return {
    subscribe,
    refresh,
  };
};

const _globalStore = createStore();

// Proxy handler to allow access like $globalStore.time_types
const proxyHandler: ProxyHandler<StoreState> = {
  get(target, prop: string | symbol) {
    if (prop in target.collections) {
      return target.collections[prop as CollectionName].items;
    }
    return target[prop as keyof StoreState];
  },
};

// Wrapped store that provides access to the collections directly
const wrappedStore: Readable<WrappedStoreValue> & { refresh: typeof _globalStore.refresh } = {
  subscribe: (run: Subscriber<WrappedStoreValue>, invalidate?: Invalidator<WrappedStoreValue>) => {
    return _globalStore.subscribe(
      (value) => run(new Proxy(value, proxyHandler) as unknown as WrappedStoreValue),
      invalidate as Invalidator<StoreState> | undefined,
    );
  },
  refresh: _globalStore.refresh,
};

export const globalStore = wrappedStore;
