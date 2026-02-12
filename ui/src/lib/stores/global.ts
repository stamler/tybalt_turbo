/**
 * The global store is used to load data that is used in multiple places in the
 * app.
 */

import type { UserPoPermissionDataResponse } from "$lib/pocketbase-types";
import { writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
import type { ClientResponseError } from "pocketbase";
import type { Readable, Subscriber } from "svelte/store";

interface ErrorMessage {
  message: string;
  id: string;
}

interface StoreState {
  isLoading: boolean;
  claims: string[];
  showAllUi: boolean;
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
  profile: {
    default_division: string;
    default_role: string;
    default_branch: string;
    maxAge: number;
    lastRefresh: Date;
  };
  error: ClientResponseError | null;
  errorMessages: ErrorMessage[];
}

const createStore = () => {
  type UserDefaultsResponse = {
    default_division: string;
    default_role: string;
    default_branch: string;
  };
  type MaybeAbortError = Partial<ClientResponseError> & {
    isAbort?: boolean;
    originalError?: { name?: string };
    message?: string;
    status?: number;
  };

  let userPoPermissionDataPromise: Promise<void> | null = null;
  let userDefaultsPromise: Promise<void> | null = null;

  const isAutoCancelled = (error: unknown): boolean => {
    const err = error as MaybeAbortError;
    if (err?.isAbort) return true;
    if (err?.originalError?.name === "AbortError") return true;
    return err?.status === 0 && (err?.message ?? "").toLowerCase().includes("aborted");
  };

  const initialShowAll = (() => {
    try {
      const v = localStorage.getItem("tybalt_showAllUi");
      return v === "true";
    } catch {
      return false;
    }
  })();

  const { subscribe, update } = writable<StoreState>({
    isLoading: false,
    claims: [],
    showAllUi: initialShowAll,
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
    profile: {
      default_division: "",
      default_role: "",
      default_branch: "",
      maxAge: 3600 * 1000,
      lastRefresh: new Date(0),
    },
    error: null,
    errorMessages: [],
  });

  const loadUserPoPermissionData = async () => {
    if (userPoPermissionDataPromise) return userPoPermissionDataPromise;
    userPoPermissionDataPromise = (async () => {
      try {
        const userId = get(authStore)?.model?.id || "";
        if (!userId) return;
        const userPoPermissionData = await pb
          .collection("user_po_permission_data")
          .getFullList<UserPoPermissionDataResponse>({
            filter: pb.filter("id={:userId}", { userId }),
            requestKey: null, // Avoid PocketBase auto-cancel across rapid refreshes.
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
        if (isAutoCancelled(error)) return;
        const typedErr = error as ClientResponseError;
        console.error("Error loading user po permission data:", typedErr);
      }
    })().finally(() => {
      userPoPermissionDataPromise = null;
    });
    return userPoPermissionDataPromise;
  };

  const loadUserDefaults = async () => {
    if (userDefaultsPromise) return userDefaultsPromise;
    userDefaultsPromise = (async () => {
      try {
        const defaults = (await pb.send("/api/users/defaults", {
          method: "GET",
          requestKey: null, // Avoid PocketBase auto-cancel across rapid refreshes.
        })) as UserDefaultsResponse;

        update((state) => {
          return {
            ...state,
            profile: {
              default_division: defaults.default_division ?? "",
              default_role: defaults.default_role ?? "",
              default_branch: defaults.default_branch ?? "",
              maxAge: state.profile.maxAge,
              lastRefresh: new Date(),
            },
          };
        });
      } catch (error: unknown) {
        if (isAutoCancelled(error)) return;
        const typedErr = error as ClientResponseError;
        console.error("Error loading user defaults:", typedErr);
      }
    })().finally(() => {
      userDefaultsPromise = null;
    });
    return userDefaultsPromise;
  };

  const refresh = async () => {
    // refresh() should no-op if the user is not logged in
    if (!get(authStore)?.isValid) {
      console.log("User is not logged in, skipping refresh");
      update((state) => {
        return {
          ...state,
          profile: {
            default_division: "",
            default_role: "",
            default_branch: "",
            maxAge: state.profile.maxAge,
            lastRefresh: new Date(0),
          },
        };
      });
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

      if (now.getTime() - state.profile.lastRefresh.getTime() >= state.profile.maxAge) {
        loadUserDefaults();
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

  const toggleShowAllUi = () => {
    update((state) => {
      const next = !state.showAllUi;
      try {
        localStorage.setItem("tybalt_showAllUi", String(next));
      } catch {
        // noop
      }
      return { ...state, showAllUi: next };
    });
  };

  return {
    subscribe,
    refresh,
    addError,
    dismissError,
    toggleShowAllUi,
  };
};

const _globalStore = createStore();

// Proxy handler to allow access like $globalStore.claims
const proxyHandler: ProxyHandler<StoreState> = {
  get(target, prop: string | symbol) {
    if (prop === "claims") {
      return target.user_po_permission_data.claims;
    }
    return target[prop as keyof StoreState];
  },
};

// Wrapped store that provides access to the collections directly
const wrappedStore: Readable<StoreState> & {
  refresh: typeof _globalStore.refresh;
  addError: typeof _globalStore.addError;
  dismissError: typeof _globalStore.dismissError;
  toggleShowAllUi: typeof _globalStore.toggleShowAllUi;
} = {
  subscribe: (run: Subscriber<StoreState>, invalidate?: () => void) => {
    return _globalStore.subscribe(
      (value) => run(new Proxy(value, proxyHandler) as unknown as StoreState),
      invalidate,
    );
  },
  refresh: _globalStore.refresh,
  addError: _globalStore.addError,
  dismissError: _globalStore.dismissError,
  toggleShowAllUi: _globalStore.toggleShowAllUi,
};

export const globalStore = wrappedStore;
