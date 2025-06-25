import { get, writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { type UnsubscribeFunc } from "pocketbase";
import MiniSearch from "minisearch";
import type { RecordFullListOptions, RecordModel } from "pocketbase";
import type { Options } from "minisearch";
import type { BaseSystemFields } from "$lib/pocketbase-types";
import { emitCollectionEvent } from "./collectionEvents";

// Define the type for our store data
type DataStore<T> = {
  items: T[];
  index: MiniSearch<T> | null;
  loading: boolean;
  error: string | null;
  initialized: boolean;
};

export function createCollectionStore<T extends BaseSystemFields>(
  collectionName: string,
  queryOptions: RecordFullListOptions,
  indexOptions: Options<T>,
  onCreate?: (item: RecordModel) => Promise<void> | undefined,
  onUpdate?: (item: RecordModel) => Promise<void> | undefined,
  proxyCollectionName?: string, // if provided, watch this collection for subscription events
  fetchAll?: () => Promise<T[]>, // optional custom function to fetch all items – allows bypassing PocketBase when we need to avoid the N+1 queries
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
    // If a custom fetchAll function is supplied use it, otherwise fall back to PocketBase getFullList
    const items: T[] = fetchAll
      ? await fetchAll()
      : await pb.collection(collectionName).getFullList<T>(queryOptions);
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

    unsubscribeFunc = await pb.collection(proxyCollectionName ?? collectionName).subscribe("*", async (e) => {
      store.update((state) => ({ ...state, loading: true }));
      try {
        if (e.action === "create") {
          if (onCreate !== undefined) {
            await onCreate(e.record);
            emitCollectionEvent(collectionName, "create", e.record.id);
          }
        } else if (e.action === "update") {
          if (onUpdate !== undefined) {
            await onUpdate(e.record);
            emitCollectionEvent(collectionName, "update", e.record.id);
          }
        } else if (e.action === "delete") {
          // Remove the deleted record from the store and discard it from the index
          store.update((state) => ({
            ...state,
            items: state.items.filter((i) => i.id !== e.record.id),
            index: state.index?.discard(e.record.id) || state.index,
          }));
          emitCollectionEvent(collectionName, "delete", e.record.id);
        }
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
    // Expose update method for callbacks to use
    update: store.update,

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
    refresh: async (id?: string) => {
      store.update((state) => ({ ...state, loading: true }));
      if (id !== undefined) {
        // Just call the onUpdate callback for this item
        if (onUpdate !== undefined) {
          await onUpdate({ id } as RecordModel);
        }
      } else {
        // refresh all items
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
    }
    },

    // Clean up subscription when the store is no longer needed
    unsubscribe: () => {
      if (unsubscribeFunc) {
        unsubscribeFunc();
        unsubscribeFunc = null;
      }
      store.update((state) => ({ ...state, initialized: false, items: [], index: null, loading: false, error: null }));
    },
  };
}
