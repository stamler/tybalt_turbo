/**
 * App configuration store
 *
 * Loads configuration values from the app_config collection and provides
 * derived stores for specific feature flags like jobsEditingEnabled.
 */

import { writable, derived, get } from "svelte/store";
import { pb } from "$lib/pocketbase";
import type { UnsubscribeFunc } from "pocketbase";
import type { AppConfigResponse } from "$lib/pocketbase-types";

interface AppConfigState {
  items: AppConfigResponse[];
  loading: boolean;
  initialized: boolean;
  error: string | null;
}

const store = writable<AppConfigState>({
  items: [],
  loading: false,
  initialized: false,
  error: null,
});

let unsubscribeFunc: UnsubscribeFunc | null = null;

/**
 * Initialize the store by loading all config records and setting up
 * realtime subscription for updates.
 */
async function init() {
  // Check authentication before proceeding
  if (!pb.authStore.token || !pb.authStore.record) return;

  // If already initialized, return (idempotent)
  if (get(store).initialized) return;

  store.update((state) => ({ ...state, loading: true }));

  try {
    // Load all config records
    const items = await pb.collection("app_config").getFullList<AppConfigResponse>({
      requestKey: "app_config_init",
    });

    store.update((state) => ({
      ...state,
      items,
      loading: false,
      initialized: true,
      error: null,
    }));

    // Set up realtime subscription
    if (unsubscribeFunc) {
      unsubscribeFunc();
    }

    unsubscribeFunc = await pb.collection("app_config").subscribe("*", async (e) => {
      if (e.action === "create") {
        store.update((state) => ({
          ...state,
          items: [...state.items, e.record as unknown as AppConfigResponse],
        }));
      } else if (e.action === "update") {
        store.update((state) => ({
          ...state,
          items: state.items.map((item) =>
            item.id === e.record.id ? (e.record as unknown as AppConfigResponse) : item,
          ),
        }));
      } else if (e.action === "delete") {
        store.update((state) => ({
          ...state,
          items: state.items.filter((item) => item.id !== e.record.id),
        }));
      }
    });
  } catch (error) {
    store.update((state) => ({
      ...state,
      loading: false,
      initialized: false,
      error: error instanceof Error ? error.message : "Failed to load config",
    }));
  }
}

/**
 * Clean up subscription when store is no longer needed
 */
function unsubscribe() {
  if (unsubscribeFunc) {
    unsubscribeFunc();
    unsubscribeFunc = null;
  }
  store.update((state) => ({
    ...state,
    initialized: false,
    items: [],
    loading: false,
    error: null,
  }));
}

/**
 * Helper to get a config value by domain key
 */
function getConfigValue(items: AppConfigResponse[], domainKey: string): unknown | null {
  const config = items.find((item) => item.key === domainKey);
  if (!config) return null;
  return config.value;
}

/**
 * Helper to get a boolean property from a domain config
 * Returns defaultValue if domain or property doesn't exist
 */
function getConfigBool(
  items: AppConfigResponse[],
  domainKey: string,
  property: string,
  defaultValue: boolean,
): boolean {
  const value = getConfigValue(items, domainKey);
  if (value === null || typeof value !== "object") return defaultValue;

  const obj = value as Record<string, unknown>;
  if (!(property in obj)) return defaultValue;

  const propValue = obj[property];
  if (typeof propValue !== "boolean") return defaultValue;

  return propValue;
}

// Derived store for jobs editing enabled
// Reads from app_config where key="jobs", checks value.create_edit_absorb
// Defaults to true (fail-open) if config is missing
export const jobsEditingEnabled = derived(store, ($store) => {
  return getConfigBool($store.items, "jobs", "create_edit_absorb", true);
});

// Derived store for expenses/PO/vendor editing enabled
// Reads from app_config where key="expenses", checks value.create_edit_absorb
// Defaults to true (fail-open) if config is missing
export const expensesEditingEnabled = derived(store, ($store) => {
  return getConfigBool($store.items, "expenses", "create_edit_absorb", true);
});

// Export the store with init and unsubscribe methods
export const appConfig = {
  subscribe: store.subscribe,
  init,
  unsubscribe,
};
