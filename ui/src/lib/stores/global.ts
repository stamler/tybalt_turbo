/**
 * The global store is used to load data that is used in multiple places in the
 * app.
 */

import type { UserPoPermissionDataResponse, ProfilesResponse } from "$lib/pocketbase-types";
import { writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { authStore } from "$lib/stores/auth";
import { get } from "svelte/store";
import { ClientResponseError } from "pocketbase";
import type { Readable, Subscriber } from "svelte/store";

interface ErrorMessage {
  message: string;
  id: string;
}

interface StoreState {
  isLoading: boolean;
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
    id: string;
    default_division: string;
    maxAge: number;
    lastRefresh: Date;
    unsubscribe?: () => void;
  };
  error: ClientResponseError | null;
  errorMessages: ErrorMessage[];
}

const createStore = () => {
  const { subscribe, update } = writable<StoreState>({
    isLoading: false,
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
      id: "",
      default_division: "",
      maxAge: 3600 * 1000,
      lastRefresh: new Date(0),
      unsubscribe: undefined,
    },
    error: null,
    errorMessages: [],
  });

  const loadUserPoPermissionData = async () => {
    try {
      const userId = get(authStore)?.model?.id || "";
      const userPoPermissionData = await pb
        .collection("user_po_permission_data")
        .getFullList<UserPoPermissionDataResponse>({
          filter: pb.filter("id={:userId}", { userId }),
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
      const typedErr = error as ClientResponseError;
      console.error("Error loading user po permission data:", typedErr);
    }
  };

  const loadUserProfile = async () => {
    try {
      const userId = get(authStore)?.model?.id || "";
      if (!userId) return;

      const profile = (await pb
        .collection("profiles")
        .getFirstListItem<ProfilesResponse>(
          pb.filter("uid={:uid}", { uid: userId }),
        )) as ProfilesResponse;

      // update state and (re)subscribe to realtime changes
      update((state) => {
        // clean up previous subscription if switching users
        if (state.profile.unsubscribe) {
          try {
            state.profile.unsubscribe();
          } catch {
            // noop
          }
        }

        // subscribe to this profile record for realtime changes
        let unsubPromise: Promise<() => void> | undefined = undefined;
        try {
          unsubPromise = pb.collection("profiles").subscribe(profile.id, (e) => {
            const newDefaultDivision =
              (e?.record as unknown as ProfilesResponse)?.default_division ??
              state.profile.default_division;
            update((s) => ({
              ...s,
              profile: {
                ...s.profile,
                default_division: newDefaultDivision,
              },
            }));
          });
        } catch {
          // noop
        }
        const unsubscribe = () => {
          if (!unsubPromise) return;
          unsubPromise
            .then((fn) => {
              try {
                fn();
              } catch {
                // noop
              }
            })
            .catch(() => {
              // noop
            });
        };

        return {
          ...state,
          profile: {
            id: profile.id,
            default_division: profile.default_division ?? "",
            maxAge: state.profile.maxAge,
            lastRefresh: new Date(),
            unsubscribe,
          },
        };
      });
    } catch (error: unknown) {
      const typedErr = error as ClientResponseError;
      console.error("Error loading user profile:", typedErr);
    }
  };

  const refresh = async () => {
    // refresh() should no-op if the user is not logged in
    if (!get(authStore)?.isValid) {
      console.log("User is not logged in, skipping refresh");
      // also clear profile subscription if any
      update((state) => {
        if (state.profile.unsubscribe) {
          try {
            state.profile.unsubscribe();
          } catch {
            // noop
          }
        }
        return {
          ...state,
          profile: {
            id: "",
            default_division: "",
            maxAge: state.profile.maxAge,
            lastRefresh: new Date(0),
            unsubscribe: undefined,
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
        loadUserProfile();
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

  return {
    subscribe,
    refresh,
    addError,
    dismissError,
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
};

export const globalStore = wrappedStore;
