import { get, writable } from 'svelte/store';
import type { JobsResponse } from '$lib/pocketbase-types';
import { pb } from '$lib/pocketbase';
import { type UnsubscribeFunc } from "pocketbase";
import MiniSearch from 'minisearch';

const collectionName = "jobs";

// Define the type for our store data
type DataStore = {
  items: JobsResponse[];
  index: MiniSearch<JobsResponse> | null;
  loading: boolean;
  error: string | null;
  initialized: boolean;
};

// Create the store
const store = writable<DataStore>({
  items: [],
  index: null,
  loading: false,
  error: null,
  initialized: false
});

// Initialize the store with data
async function initializeStore() {
  const items = await pb.collection(collectionName).getFullList<JobsResponse>({
    expand: "categories_via_job,client",
    sort: "-number",
    requestKey: "job",
  });
  const itemsIndex = new MiniSearch<JobsResponse>({
    fields: ["id", "number", "description", "client"],
    storeFields: ["id", "number", "description", "client"],
    extractField: (document, fieldName) => {
      if (fieldName === "client") {
        return document.expand?.client?.name ?? "";
      }
      return document[fieldName as keyof JobsResponse] as string;
    },
  });
  itemsIndex.addAll(items as JobsResponse[]);
  store.update(state => ({
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
  
  unsubscribeFunc = await pb.collection(collectionName).subscribe('*', async () => {
    // TODO (EFFICIENCY): there's probably a way to make this SIGNIFICANTLY more
    // efficient by refactoring the jobs endpoint to allow us to specify a job id
    // and get a single job then just update that one job in the store. This works
    // for now.
    try {
      await initializeStore();
    } catch (error) {
      // handle error, ensure initialized is false
      store.update(state => ({ ...state, loading: false, initialized: false, error: error instanceof Error ? error.message : 'Failed to load items' }));
    }
  });
}

export const collectionStore = {
  subscribe: store.subscribe,
  
  // Initialize the store and subscription (call this when the store is first used)
  init: async () => {
    // if the store is already initialized, return so the function is idempotent
    if (get(store).initialized) return;
    
    store.update(state => ({ ...state, loading: true }));
    try {
      await initializeStore();
      await setupSubscription();
      store.update(state => ({ ...state, loading: false, initialized: true }));    
    } catch (error) {
      // handle error, ensure initialized is false
      store.update(state => ({ ...state, loading: false, initialized: false, error: error instanceof Error ? error.message : 'Failed to load items' }));
    }
  },
  
  // Refresh the data manually if needed
  refresh: async () => {
    store.update(state => ({ ...state, loading: true }));
    try {
      await initializeStore();
      store.update(state => ({ ...state, loading: false }));
    } catch (error) {
      // handle error, ensure initialized is false
      store.update(state => ({ ...state, loading: false, error: error instanceof Error ? error.message : 'Failed to load items' }));
    }
  },
  
  // Clean up subscription when the store is no longer needed
  unsubscribe: () => {
    if (unsubscribeFunc) {
      unsubscribeFunc();
      unsubscribeFunc = null;
    }
  }
};
