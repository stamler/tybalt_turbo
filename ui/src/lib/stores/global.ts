/**
 * The global store is used to load data that is used in multiple places in the
 * app.
 */

import { writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { authStore } from "$lib/stores/auth";
import type {
  UserClaimsSummaryResponse,
  UserPoApproverProfileResponse,
} from "$lib/pocketbase-types";
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
  user_claims_summary: {
    id: string;
    claims: string[];
    maxAge: number;
    lastRefresh: Date;
  };
  user_po_approver_profile: {
    id: string;
    max_amount: number;
    project_max: number;
    sponsorship_max: number;
    staff_and_social_max: number;
    media_and_event_max: number;
    computer_max: number;
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

  let userClaimsSummaryPromise: Promise<void> | null = null;
  let userPoApproverProfilePromise: Promise<void> | null = null;
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
    user_claims_summary: {
      id: "",
      claims: [],
      maxAge: 3600 * 1000,
      lastRefresh: new Date(0),
    },
    user_po_approver_profile: {
      id: "",
      max_amount: 0,
      project_max: 0,
      sponsorship_max: 0,
      staff_and_social_max: 0,
      media_and_event_max: 0,
      computer_max: 0,
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

  const loadUserClaimsSummary = async () => {
    if (userClaimsSummaryPromise) return userClaimsSummaryPromise;
    userClaimsSummaryPromise = (async () => {
      try {
        const userId = get(authStore)?.model?.id || "";
        if (!userId) return;

        const response = await pb
          .collection("user_claims_summary")
          .getOne<UserClaimsSummaryResponse>(userId, {
          requestKey: null,
        });

        update((state) => ({
          ...state,
          user_claims_summary: {
            id: response.id ?? "",
            claims: response.claims ?? [],
            lastRefresh: new Date(),
            maxAge: state.user_claims_summary.maxAge,
          },
        }));
      } catch (error: unknown) {
        if (isAutoCancelled(error)) return;
        const typedErr = error as ClientResponseError;
        if (typedErr?.status === 404) {
          update((state) => ({
            ...state,
            user_claims_summary: {
              id: "",
              claims: [],
              lastRefresh: new Date(),
              maxAge: state.user_claims_summary.maxAge,
            },
          }));
          return;
        }
        console.error("Error loading user claims summary:", typedErr);
      }
    })().finally(() => {
      userClaimsSummaryPromise = null;
    });
    return userClaimsSummaryPromise;
  };

  const loadUserPoApproverProfile = async () => {
    if (userPoApproverProfilePromise) return userPoApproverProfilePromise;
    userPoApproverProfilePromise = (async () => {
      try {
        const userId = get(authStore)?.model?.id || "";
        if (!userId) return;

        const response = await pb
          .collection("user_po_approver_profile")
          .getOne<UserPoApproverProfileResponse>(userId, {
            requestKey: null,
          });

        update((state) => ({
          ...state,
          user_po_approver_profile: {
            id: response.id ?? "",
            max_amount: response.max_amount ?? 0,
            project_max: response.project_max ?? 0,
            sponsorship_max: response.sponsorship_max ?? 0,
            staff_and_social_max: response.staff_and_social_max ?? 0,
            media_and_event_max: response.media_and_event_max ?? 0,
            computer_max: response.computer_max ?? 0,
            divisions: response.divisions ?? [],
            claims: response.claims ?? [],
            lastRefresh: new Date(),
            maxAge: state.user_po_approver_profile.maxAge,
          },
        }));
      } catch (error: unknown) {
        if (isAutoCancelled(error)) return;
        const typedErr = error as ClientResponseError;
        if (typedErr?.status === 404) {
          update((state) => ({
            ...state,
            user_po_approver_profile: {
              id: "",
              max_amount: 0,
              project_max: 0,
              sponsorship_max: 0,
              staff_and_social_max: 0,
              media_and_event_max: 0,
              computer_max: 0,
              divisions: [],
              claims: [],
              lastRefresh: new Date(),
              maxAge: state.user_po_approver_profile.maxAge,
            },
          }));
          return;
        }
        console.error("Error loading user PO approver profile:", typedErr);
      }
    })().finally(() => {
      userPoApproverProfilePromise = null;
    });
    return userPoApproverProfilePromise;
  };

  const loadUserDefaults = async () => {
    if (userDefaultsPromise) return userDefaultsPromise;
    userDefaultsPromise = (async () => {
      try {
        const defaults = (await pb.send("/api/users/defaults", {
          method: "GET",
          requestKey: null,
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
    if (!get(authStore)?.isValid) {
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
        now.getTime() - state.user_claims_summary.lastRefresh.getTime() >=
        state.user_claims_summary.maxAge
      ) {
        loadUserClaimsSummary();
      }

      if (
        now.getTime() - state.user_po_approver_profile.lastRefresh.getTime() >=
        state.user_po_approver_profile.maxAge
      ) {
        loadUserPoApproverProfile();
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

const proxyHandler: ProxyHandler<StoreState> = {
  get(target, prop: string | symbol) {
    if (prop === "claims") {
      return target.user_claims_summary.claims;
    }
    return target[prop as keyof StoreState];
  },
};

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
