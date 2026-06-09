import { pb } from "$lib/pocketbase";
import { AUTH_SESSION_EXPIRED_EVENT, authStore } from "$lib/stores/auth";
import { AUTH_CONFIG } from "$lib/config";
import { jobs } from "$lib/stores/jobs";
import { vendors } from "$lib/stores/vendors";
import { clients } from "$lib/stores/clients";
import { divisions } from "$lib/stores/divisions";
import { timeTypes } from "$lib/stores/time_types";
import { globalStore } from "$lib/stores/global";

const allStores = [jobs, vendors, clients, divisions, timeTypes];

type AuthIdentityRecord = {
  id?: string;
  collectionId?: string;
  collectionName?: string;
};

function authIdentity(record: AuthIdentityRecord | null | undefined): string {
  if (!record?.id) return "";
  return `${record.collectionId ?? record.collectionName ?? "auth"}:${record.id}`;
}

/**
 * CLIENT-SIDE AUTH INITIALIZATION
 * ===============================
 *
 * This file runs ONCE when the SvelteKit app first loads in the browser.
 * It sets up the entire auth system and keeps it synchronized.
 *
 * WHAT THIS FILE DOES:
 * 1. Sets up app wake/activity checks across the browser
 * 2. Attempts to refresh existing or near-expiry tokens before users hit failures
 * 3. Initializes the Svelte auth store with current state
 * 4. Keeps the store synchronized with PocketBase auth changes
 *
 * WAKE/ACTIVITY CHECKS:
 * - Listens for selected user interactions, focus, visibility, and online events
 * - These events do not enforce an idle timeout
 * - They simply give the auth store a chance to restart a missed timer or
 *   refresh a near-expiry token after a tab wakes up or comes back online
 *
 * TOKEN REFRESH:
 * - Validate existing browser auth state immediately on app startup
 * - Refresh authenticated sessions shortly before the JWT expires
 * - Attempt one last refresh before protected API requests
 * - Probe non-auth 401 responses with authRefresh so permission failures do not
 *   log out users with otherwise valid sessions
 * - Clear expired auth state so the layout can send the user to login
 *
 * STORE SYNCHRONIZATION:
 * - PocketBase has its own auth store (pb.authStore)
 * - We maintain a separate Svelte store (authStore) for reactive UI updates
 * - This onChange callback keeps them synchronized
 */

// STEP 1: Set up centralized request auth handling before any app requests fire.
authStore.setupRequestAuthGuard();

// STEP 2: Set up app-wide wake/activity checks.
// These events do not decide whether the user is "active enough" to stay signed
// in. The app intentionally has no idle timeout. They only give the auth store a
// timely chance to refresh when the browser tab wakes, focuses, reconnects, or
// receives input.
AUTH_CONFIG.SESSION_CHECK_EVENTS.forEach((event) => {
  document.addEventListener(event, authStore.updateActivity, true);
});

document.addEventListener("visibilitychange", () => {
  if (!document.hidden) {
    authStore.updateActivity();
  }
});
window.addEventListener("focus", authStore.updateActivity);
window.addEventListener("online", authStore.updateActivity);
window.addEventListener(AUTH_SESSION_EXPIRED_EVENT, (event) => {
  const message =
    event instanceof CustomEvent && typeof event.detail?.message === "string"
      ? event.detail.message
      : "Your session expired. Sign in again to continue.";
  globalStore.addError(message, { source: "auth-session" });
});

// STEP 3: Initialize the Svelte store with current auth state
// This makes the auth state immediately available to all Svelte components
authStore.refresh();

// STEP 4: Handle existing token on app startup
// If user has a token from a previous session, refresh it to ensure it is
// still valid and to extend its lifetime.
if (pb.authStore.token && pb.authStore.record) {
  authStore.refreshAuth().then((success) => {
    if (success) {
      // Token refresh succeeded - start the expiry-aware refresh system.
      // This keeps the user logged in as long as PocketBase auth can continue
      // refreshing, regardless of local app inactivity.
      authStore.setupTokenRefresh();
      // load global user-scoped data (profile, permissions)
      globalStore.refresh();
    }
    // If refresh failed, refreshAuth already cleared the invalid auth state.
  });
}

// STEP 5: Keep Svelte store synchronized with PocketBase auth changes
// This onChange callback fires whenever PocketBase auth state changes:
// - User logs in (successful OAuth2, password login, etc.)
// - User logs out (manual logout, token expiration, etc.)
// - Token gets refreshed
// - Auth state gets cleared due to errors
let previousAuthIdentity = "";
pb.authStore.onChange(() => {
  const currentAuthIdentity = authIdentity(pb.authStore.record);
  const authIdentityChanged = currentAuthIdentity !== previousAuthIdentity;
  previousAuthIdentity = currentAuthIdentity;

  // Update our Svelte store to match PocketBase state
  authStore.refresh();

  // If user is now authenticated, set up the refresh timer
  // If user is not authenticated, setupTokenRefresh() will clear any existing timer
  if (pb.authStore.token && pb.authStore.record) {
    authStore.setupTokenRefresh();

    if (!authIdentityChanged) return;

    globalStore.clearErrors({ source: "auth-session" });

    // initialize stores
    /*
     * === Asynchronous init() calls ===
     * Each `store.init()` returns a Promise because it fetches the initial
     * data set from PocketBase and then registers a realtime subscription.
     *
     * The loop below intentionally _does not_ await these Promises; the
     * operations are "fire-and-forget" so the UI can remain responsive while
     * each collection loads in the background.
     *
     * If you later decide that subsequent logic (for example showing the
     * application shell) must wait until _all_ collections are fully loaded,
     * you have two straightforward options:
     *
     * 1. Make the `onChange` callback itself `async` and `await` them:
     *
     *        pb.authStore.onChange(async () => {
     *          authStore.refresh();
     *          if (authenticated) {
     *            await Promise.all(allStores.map(s => s.init()));
     *          }
     *        }, true);
     *
     * 2. Keep the callback synchronous and wrap the awaited work in an IIFE:
     *
     *        (async () => {
     *          await Promise.all(allStores.map(s => s.init()));
     *        })();
     *
     * Both variants will block until every collection has completed its
     * initial fetch, allowing you to toggle a global loading indicator or
     * similar UX affordance.
     */
    allStores.forEach((store) => store.init());
    // refresh global store (profile, permissions)
    globalStore.refresh();
  } else if (authIdentityChanged) {
    // clear the stores
    allStores.forEach((store) => store.unsubscribe());
  }
}, true); // The 'true' parameter means this callback also fires immediately with current state
