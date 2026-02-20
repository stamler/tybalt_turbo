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

// Set up PocketBase realtime subscription
let unsubscribeFunc: UnsubscribeFunc | null = null;

async function setupSubscription() {
  if (unsubscribeFunc) {
    unsubscribeFunc(); // Clean up existing subscription
  }

  const subTaskId = "timesheets-sub";
  unsubscribeFunc = await pb.collection("time_sheets").subscribe("*", async () => {
    // reload all the tallies. We can't specify a specific tally because there isn't a
    // time_sheets_augmented collection, but rather just an endpoint that returns all
    // the tallies.

    // TODO (EFFICIENCY): there's probably a way to make this SIGNIFICANTLY more
    // efficient by refactoring the time_sheets_tallies endpoint to allow us to
    // specify a timesheet id and get a single tally then just update that one
    // tally in the store. This works for now.
    tasks.startTask({ id: subTaskId, message: "Updating timesheets" });
    await initializeStore();
    tasks.endTask(subTaskId);
  });
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
    if (unsubscribeFunc) {
      unsubscribeFunc();
      unsubscribeFunc = null;
    }
  },
};
