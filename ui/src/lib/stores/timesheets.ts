import { writable } from "svelte/store";
import type { TimeSheetTallyQueryRow } from "$lib/utilities";
import { pb } from "$lib/pocketbase";
import { type UnsubscribeFunc } from "pocketbase";
import { tasks } from "./tasks";

// TODO: This could be changed to a readable store rather than a writable store.
// since the data is never updated outside of the store.

// Define the type for our store data
type DataStore = {
  tallies: TimeSheetTallyQueryRow[];
  loading: boolean;
  error: string | null;
  initialized: boolean;
};

// Create the store
const store = writable<DataStore>({
  tallies: [],
  loading: false,
  error: null,
  initialized: false,
});

// Initialize the store with data
async function initializeStore() {
  try {
    const taskId = "timesheets-load";
    tasks.startTask({ id: taskId, message: "Loading timesheets" });
    store.update((state) => ({ ...state, loading: true, error: null }));
    const tallies = await pb.send("/api/time_sheets/tallies", {
      requestKey: "tallies",
    });
    store.update((state) => ({
      ...state,
      tallies,
      loading: false,
      initialized: true,
    }));
    tasks.endTask(taskId);
  } catch (error) {
    store.update((state) => ({
      ...state,
      error: error instanceof Error ? error.message : "Failed to load time sheets",
      loading: false,
      initialized: true,
    }));
    tasks.endTask("timesheets-load");
  }
}

// Set up PocketBase realtime subscription for time_sheets updates.
let unsubscribeFuncs: UnsubscribeFunc[] = [];

async function setupSubscription() {
  unsubscribeFuncs.forEach((unsubscribe) => unsubscribe());
  unsubscribeFuncs = [];

  const subTaskId = "timesheets-sub";
  const refreshTallies = async () => {
    tasks.startTask({ id: subTaskId, message: "Updating timesheets" });
    await initializeStore();
    tasks.endTask(subTaskId);
  };

  try {
    unsubscribeFuncs = [await pb.collection("time_sheets").subscribe("*", refreshTallies)];
  } catch (error) {
    console.warn("Failed to subscribe to time_sheets realtime updates:", error);
    unsubscribeFuncs.forEach((unsubscribe) => unsubscribe());
    unsubscribeFuncs = [];
  }
}

// Track initialization state
let initializationPromise: Promise<void> | null = null;

export const timesheets = {
  subscribe: store.subscribe,

  // Initialize the store and subscription (call this when the store is first used)
  init: async () => {
    // Check authentication before proceeding - noop if not authenticated
    if (!pb.authStore.token || !pb.authStore.record) return;

    if (initializationPromise) {
      return initializationPromise; // Return existing promise if already initializing
    }

    initializationPromise = (async () => {
      await initializeStore();
      await setupSubscription();
    })();

    return initializationPromise;
  },

  // Refresh the data manually if needed
  refresh: async () => {
    store.update((state) => ({ ...state, loading: true }));
    await initializeStore();
  },

  // Clean up subscription when the store is no longer needed
  unsubscribe: () => {
    unsubscribeFuncs.forEach((unsubscribe) => unsubscribe());
    unsubscribeFuncs = [];
  },
};
