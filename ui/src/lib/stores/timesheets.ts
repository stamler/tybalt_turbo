import { writable } from 'svelte/store';
import type { TimeSheetTallyQueryRow } from '$lib/utilities';
import { pb } from '$lib/pocketbase';
import { type UnsubscribeFunc } from "pocketbase";

// Define the type for our store data
type DataStore = {
  tallies: TimeSheetTallyQueryRow[];
  loading: boolean;
  error: string | null;
};

// Create the store
const store = writable<DataStore>({
  tallies: [],
  loading: true,
  error: null
});

// Initialize the store with data
async function initializeStore() {
  try {
    const tallies = await pb.send("/api/time_sheets/tallies", {
      requestKey: "tallies",
    });
    store.update(state => ({
      ...state,
      tallies,
      loading: false
    }));
  } catch (error) {
    store.update(state => ({
      ...state,
      error: error instanceof Error ? error.message : 'Failed to load time sheets',
      loading: false
    }));
  }
}

// Set up PocketBase realtime subscription
let unsubscribeFunc: UnsubscribeFunc;

async function setupSubscription() {
  unsubscribeFunc = await pb.collection('time_sheets').subscribe('*', async () => {
    // reload all the tallies. We can't specify a specific tally because there isn't a
    // time_sheets_augmented collection, but rather just an endpoint that returns all
    // the tallies.

    // TODO (EFFICIENCY): there's probably a way to make this SIGNIFICANTLY more
    // efficient by refactoring the time_sheets_tallies endpoint to allow us to
    // specify a timesheet id and get a single tally then just update that one
    // tally in the store. This works for now.
    await initializeStore();
  });
}

// Initialize the store and subscription
await initializeStore();
await setupSubscription();

export const timesheets = {
  subscribe: store.subscribe,
  
  // Refresh the data manually if needed
  refresh: async () => {
    store.update(state => ({ ...state, loading: true }));
    await initializeStore();
  },
  
  // Clean up subscription when the store is no longer needed
  unsubscribe: () => {
    unsubscribeFunc();
  }
};
