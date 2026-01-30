import { get, writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { type UnsubscribeFunc } from "pocketbase";
import MiniSearch from "minisearch";
import type { RecordFullListOptions, RecordModel } from "pocketbase";
import type { Options } from "minisearch";
import type { BaseSystemFields } from "$lib/pocketbase-types";
import { emitCollectionEvent } from "./collectionEvents";
import { tasks } from "./tasks";

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
  enableAbsorbSubscription = false, // whether to listen for <collection>/absorb_completed events
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
    // Register loading task
    const taskId = `init-${collectionName}`;
    tasks.startTask({ id: taskId, message: `Loading ${collectionName}` });
    // If a custom fetchAll function is supplied use it, otherwise fall back to PocketBase getFullList
    const items: T[] = fetchAll
      ? await fetchAll()
      : await pb.collection(collectionName).getFullList<T>(queryOptions);
    // Task completed
    tasks.endTask(taskId);
    // Merge default searchOptions (AND logic) with any provided options
    const itemsIndex = new MiniSearch<T>({
      ...indexOptions,
      searchOptions: {
        combineWith: "AND",
        ...indexOptions.searchOptions,
      },
    });
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

    const subTaskId = `sub-${collectionName}`;
    unsubscribeFunc = await pb
      .collection(proxyCollectionName ?? collectionName)
      .subscribe("*", async (e) => {
        store.update((state) => ({ ...state, loading: true }));
        tasks.startTask({ id: subTaskId, message: `Updating ${collectionName}` });
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
          tasks.endTask(subTaskId);
        } catch (error) {
          // handle error, ensure initialized is false
          store.update((state) => ({
            ...state,
            loading: false,
            initialized: false,
            error: error instanceof Error ? error.message : "Failed to load items",
          }));
          tasks.endTask(subTaskId);
        }
      });
  }

  // Refresh the data manually if needed – declared here so other helpers can reference it.
  const refresh = async (id?: string) => {
    store.update((state) => ({ ...state, loading: true }));
    const refreshId = `refresh-${collectionName}`;
    tasks.startTask({ id: refreshId, message: `Refreshing ${collectionName}` });
    if (id !== undefined) {
      if (onUpdate !== undefined) {
        await onUpdate({ id } as RecordModel);
      }
      store.update((state) => ({ ...state, loading: false }));
      tasks.endTask(refreshId);
      return;
    }

    try {
      await initializeStore();
      store.update((state) => ({ ...state, loading: false }));
      tasks.endTask(refreshId);
    } catch (error) {
      store.update((state) => ({
        ...state,
        loading: false,
        error: error instanceof Error ? error.message : "Failed to load items",
      }));
      tasks.endTask(refreshId);
    }
  };

  // Subscribe to custom "absorb_completed" events so the whole collection can refresh after a bulk absorb.
  // Separate unsubscribe function for the absorb_completed realtime channel
  let unsubscribeAbsorb: UnsubscribeFunc | null = null;
  async function setupAbsorbSubscription() {
    // Guard: only setup if enabled
    if (!enableAbsorbSubscription) return;

    // Clean up existing
    if (unsubscribeAbsorb) {
      unsubscribeAbsorb();
    }
    // The channel name is "<collection>/absorb_completed"
    const topic = `${collectionName}/absorb_completed`;

    unsubscribeAbsorb = await pb.realtime.subscribe(topic, async () => {
      await refresh();
    });
  }

  return {
    subscribe: store.subscribe,
    // Expose update method for callbacks to use
    update: store.update,

    // Initialize the store and subscription (call this when the store is first used)
    init: async () => {
      // Check authentication before proceeding - noop if not authenticated
      if (!pb.authStore.token || !pb.authStore.model) return;

      // if the store is already initialized, return so the function is idempotent
      if (get(store).initialized) return;

      store.update((state) => ({ ...state, loading: true }));
      try {
        await initializeStore();
        if (enableAbsorbSubscription) {
          await Promise.all([setupSubscription(), setupAbsorbSubscription()]);
        } else {
          await setupSubscription();
        }
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
    refresh,

    // Clean up subscription when the store is no longer needed
    unsubscribe: () => {
      if (unsubscribeFunc) {
        unsubscribeFunc();
        unsubscribeFunc = null;
      }
      if (enableAbsorbSubscription && unsubscribeAbsorb) {
        unsubscribeAbsorb();
        unsubscribeAbsorb = null;
      }
      store.update((state) => ({
        ...state,
        initialized: false,
        items: [],
        index: null,
        loading: false,
        error: null,
      }));
    },
  };
}
