import { get, writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { type UnsubscribeFunc } from "pocketbase";
import MiniSearch from "minisearch";
import type { RecordFullListOptions } from "pocketbase";
import type { Options } from "minisearch";

// Define the type for our store data
type DataStore<T> = {
  items: T[];
  index: MiniSearch<T> | null;
  loading: boolean;
  error: string | null;
  initialized: boolean;
};

export function createCollectionStore<T>(
  collectionName: string,
  queryOptions: RecordFullListOptions,
  indexOptions: Options<T>,
) {
  // Create the store
  const store = writable<DataStore<T>>({
    items: [],
    index: null,
    loading: false,
    error: null,
    initialized: false,
  });

  // Initialize the store with data
  async function initializeStore() {
    const items = await pb.collection(collectionName).getFullList<T>(queryOptions);
    const itemsIndex = new MiniSearch<T>(indexOptions);
    itemsIndex.addAll(items);
    store.update((state) => ({
      ...state,
      items,
      index: itemsIndex,
    }));
  }

  // Set up PocketBase realtime subscription
  let unsubscribeFunc: UnsubscribeFunc | null = null;

  async function setupSubscription() {
    if (unsubscribeFunc) {
      unsubscribeFunc(); // Clean up existing subscription
    }

    unsubscribeFunc = await pb.collection(collectionName).subscribe("*", async () => {
      // TODO: make this more efficient. Instead of reloading the
      // entire collection, we should just reload the single item that changed.
      // TODO: This function should be passed in as a parameter to the store
      // constructor, either as multiple parameters (one each for create, update, delete)
      // or as a single function that checks each case.
      store.update((state) => ({ ...state, loading: true }));
      try {
        await initializeStore();
        store.update((state) => ({ ...state, loading: false }));
      } catch (error) {
        // handle error, ensure initialized is false
        store.update((state) => ({
          ...state,
          loading: false,
          initialized: false,
          error: error instanceof Error ? error.message : "Failed to load items",
        }));
      }
    });
  }

  return {
    subscribe: store.subscribe,

    // Initialize the store and subscription (call this when the store is first used)
    init: async () => {
      // if the store is already initialized, return so the function is idempotent
      if (get(store).initialized) return;

      store.update((state) => ({ ...state, loading: true }));
      try {
        await initializeStore();
        await setupSubscription();
        store.update((state) => ({ ...state, loading: false, initialized: true }));
      } catch (error) {
        // handle error, ensure initialized is false
        store.update((state) => ({
          ...state,
          loading: false,
          initialized: false,
          error: error instanceof Error ? error.message : "Failed to load items",
        }));
      }
    },

    // Refresh the data manually if needed
    refresh: async () => {
      store.update((state) => ({ ...state, loading: true }));
      try {
        await initializeStore();
        store.update((state) => ({ ...state, loading: false }));
      } catch (error) {
        // handle error, ensure initialized is false
        store.update((state) => ({
          ...state,
          loading: false,
          error: error instanceof Error ? error.message : "Failed to load items",
        }));
      }
    },

    // Clean up subscription when the store is no longer needed
    unsubscribe: () => {
      if (unsubscribeFunc) {
        unsubscribeFunc();
        unsubscribeFunc = null;
      }
    },
  };
}
