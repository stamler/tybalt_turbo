import { pb } from "$lib/pocketbase";
import { authStore } from "$lib/stores/auth";
import { AUTH_CONFIG } from "$lib/config";
import { jobs } from "$lib/stores/jobs";
import { vendors } from "$lib/stores/vendors";
import { clients } from "$lib/stores/clients";

/**
 * CLIENT-SIDE AUTH INITIALIZATION
 * ===============================
 * 
 * This file runs ONCE when the SvelteKit app first loads in the browser.
 * It sets up the entire auth system and keeps it synchronized.
 * 
 * WHAT THIS FILE DOES:
 * 1. Sets up activity tracking across the entire app
 * 2. Attempts to refresh any existing token on app startup
 * 3. Initializes the Svelte auth store with current state
 * 4. Keeps the store synchronized with PocketBase auth changes
 * 
 * ACTIVITY TRACKING:
 * - Listens for user interactions (mouse, keyboard, scroll, touch) on the entire document
 * - Every interaction updates the "last activity" timestamp in the auth store
 * - This timestamp is used by the refresh timer to determine if user is active
 * 
 * TOKEN REFRESH ON STARTUP:
 * - If user has an existing token from previous session, try to refresh it immediately
 * - This ensures the token is valid and extends its lifetime
 * - If refresh succeeds, start the periodic refresh timer
 * - If refresh fails, user will need to log in again
 * 
 * STORE SYNCHRONIZATION:
 * - PocketBase has its own auth store (pb.authStore)
 * - We maintain a separate Svelte store (authStore) for reactive UI updates
 * - This onChange callback keeps them synchronized
 */

// STEP 1: Set up app-wide activity tracking
// These event listeners capture ALL user interactions across the entire SPA
// The third parameter (true) means we capture during the capturing phase,
// ensuring we catch events even if they're handled by child components
AUTH_CONFIG.ACTIVITY_EVENTS.forEach(event => {
  document.addEventListener(event, authStore.updateActivity, true);
});

// STEP 2: Handle existing token on app startup
// If user has a token from a previous session (stored in browser),
// try to refresh it to ensure it's still valid and extend its lifetime
if (pb.authStore.token && pb.authStore.model) {
  authStore.refreshAuth().then((success) => {
    if (success) {
      // Token refresh succeeded - start the periodic refresh system
      // This will keep the user logged in as long as they remain active
      authStore.setupTokenRefresh();
    }
    // If refresh failed, the authStore.refreshAuth() function already cleared
    // the invalid auth state, so user will see login screen
  });
}

// STEP 3: Initialize the Svelte store with current auth state
// This makes the auth state immediately available to all Svelte components
authStore.refresh();

// STEP 4: Keep Svelte store synchronized with PocketBase auth changes
// This onChange callback fires whenever PocketBase auth state changes:
// - User logs in (successful OAuth2, password login, etc.)
// - User logs out (manual logout, token expiration, etc.)
// - Token gets refreshed
// - Auth state gets cleared due to errors
pb.authStore.onChange(() => {
  // Update our Svelte store to match PocketBase state
  authStore.refresh();
  
  // If user is now authenticated, set up the refresh timer
  // If user is not authenticated, setupTokenRefresh() will clear any existing timer
  if (pb.authStore.token && pb.authStore.model) {
    authStore.setupTokenRefresh();
  }
}, true); // The 'true' parameter means this callback also fires immediately with current state


// initialize stores
jobs.init();
vendors.init();
clients.init();