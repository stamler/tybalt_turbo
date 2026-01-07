import { writable, get } from "svelte/store";
import { pb } from "$lib/pocketbase";
import type { AuthModel } from "pocketbase";
import { AUTH_TIMEOUTS } from "$lib/config";

/**
 * AUTH SYSTEM OVERVIEW
 * ===================
 *
 * This auth store implements activity-based token refresh for security:
 *
 * 1. ACTIVE USERS: Stay logged in indefinitely via automatic token refresh
 * 2. INACTIVE USERS: Token refresh stops after 30 minutes of inactivity
 * 3. MAXIMUM TIMEOUT: Even abandoned sessions expire within 1.5 hours max
 *
 * SECURITY MODEL:
 * - Token lifetime: 1 hour (configured in PocketBase)
 * - Refresh interval: 45 minutes (configurable in config.ts)
 * - Inactivity timeout: 30 minutes (configurable in config.ts)
 * - Activity events: mouse, keyboard, scroll, touch (configurable in config.ts)
 *
 * FLOW:
 * - User logs in → Token refresh timer starts
 * - Every 45 minutes → Check if user was active in last 30 minutes
 * - If active → Refresh token and continue
 * - If inactive → Stop refreshing, let token expire naturally
 * - User activity → Always updates the "last activity" timestamp
 */

// Define the shape of our auth state
interface AuthState {
  isValid: boolean;
  model: AuthModel;
  token: string;
}

// Create the store with a clear interface
const baseStore = writable<AuthState | null>(null);
const { subscribe, set } = baseStore;

// Timer for periodic token refresh (runs every 45 minutes by default)
let refreshTimer: number | null = null;

// Track last user activity timestamp (updated by activity events from hooks.client.ts)
let lastActivity = Date.now();

/**
 * UTILITY: Extract token expiration date
 * Returns the expiration date of the current JWT token, or null if invalid/missing
 */
function tokenExpirationDate(): Date | null {
  const token = pb.authStore.token;
  if (!token) return null;

  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;

    const payload = JSON.parse(atob(parts[1]));
    if (!payload.exp) return null;

    const expirationDate = new Date(payload.exp * 1000);
    // Check if the date is valid
    return isNaN(expirationDate.getTime()) ? null : expirationDate;
  } catch (error) {
    console.warn("Failed to parse token expiration:", error);
    return null;
  }
}

/**
 * UTILITY: Check if token is valid and not expired
 * This validates the JWT structure and checks the expiration claim
 */
function checkTokenExpiration(token: string): boolean {
  if (!token) return false;

  try {
    // JWT tokens have 3 parts separated by dots
    const parts = token.split(".");
    if (parts.length !== 3) return false;

    // Decode the payload (second part)
    const payload = JSON.parse(atob(parts[1]));

    // Check if token is expired (exp is in seconds, Date.now() is in milliseconds)
    const now = Math.floor(Date.now() / 1000);
    return payload.exp && payload.exp > now;
  } catch (error) {
    console.warn("Failed to parse token:", error);
    return false;
  }
}

/**
 * CORE: Refresh the authentication token
 * Uses PocketBase's authRefresh() API to get a new token with extended expiry
 * Returns true on success, false on failure (which clears auth state)
 */
async function refreshAuth(): Promise<boolean> {
  try {
    if (!pb.authStore.token || !pb.authStore.model) {
      return false;
    }

    // Try to refresh the token using PocketBase API
    await pb.collection("users").authRefresh();

    // PocketBase automatically updates pb.authStore on successful refresh
    // The onChange callback in hooks.client.ts will update our Svelte store
    return true;
  } catch (error) {
    console.warn("Token refresh failed:", error);
    // Clear invalid auth state - user will need to log in again
    pb.authStore.clear();
    return false;
  }
}

/**
 * ACTIVITY TRACKING: Update last activity timestamp
 * Called by event listeners in hooks.client.ts whenever user interacts with the app
 * This is how we track if the user is "active" for security purposes
 */
function updateActivity() {
  lastActivity = Date.now();
}

/**
 * CORE: Setup the activity-based token refresh system
 *
 * This is the heart of our security model:
 * 1. Sets up a timer that runs every 45 minutes (configurable)
 * 2. Each time it runs, checks if user was active in last 30 minutes (configurable)
 * 3. If active: refreshes token and continues
 * 4. If inactive: stops the timer and lets the token expire naturally
 *
 * Called when:
 * - User logs in
 * - App loads and finds existing valid token
 * - PocketBase auth state changes
 */
function setupTokenRefresh() {
  // Clear any existing timer to prevent duplicates
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }

  // Only setup refresh if we have a valid token
  if (pb.authStore.token && pb.authStore.model) {
    // Reset activity timestamp when setting up refresh (treat setup as activity)
    lastActivity = Date.now();

    // Set up the periodic refresh timer
    refreshTimer = setInterval(async () => {
      const timeSinceActivity = Date.now() - lastActivity;

      if (timeSinceActivity < AUTH_TIMEOUTS.INACTIVITY_TIMEOUT_MS) {
        // User was recently active - refresh token to keep session alive
        console.log("User active - refreshing auth token");
        const success = await refreshAuth();
        if (!success) {
          // If refresh fails (e.g., server error, invalid token), clean up
          if (refreshTimer) {
            clearInterval(refreshTimer);
            refreshTimer = null;
          }
        }
      } else {
        // User inactive too long - stop refreshing and let token expire naturally
        // This is the security feature: abandoned sessions will timeout
        console.log(
          "User inactive for",
          Math.round(timeSinceActivity / 60000),
          "minutes - stopping token refresh",
        );
        if (refreshTimer) {
          clearInterval(refreshTimer);
          refreshTimer = null;
        }
      }
    }, AUTH_TIMEOUTS.TOKEN_REFRESH_INTERVAL_MS);
  }
}

/**
 * UTILITY: Get current auth state with validation
 * Checks both token presence and expiration before returning auth state
 * If token is expired, automatically clears the auth state
 */
function getCurrentAuthState(): AuthState | null {
  if (!pb.authStore.token || !pb.authStore.model) {
    return null;
  }

  // Check if token is actually valid (not expired)
  const tokenIsValid = checkTokenExpiration(pb.authStore.token);

  if (!tokenIsValid) {
    // Token is expired, clear PocketBase auth and return null
    pb.authStore.clear();
    return null;
  }

  return {
    isValid: true, // We've verified the token is valid and not expired
    model: pb.authStore.model,
    token: pb.authStore.token,
  };
}

/**
 * AUTH ACTION: Login with Microsoft OAuth2
 * Uses PocketBase's OAuth2 flow to authenticate with Microsoft
 * On success, the onChange callback in hooks.client.ts will handle setup
 *
 * IMPORTANT: This function is intentionally NOT async!
 * Safari and Chrome may block or delay the OAuth popup if window.open is called
 * from within an async function, because it breaks the direct synchronous link
 * between the user's click and the popup. Using .then()/.catch() keeps the
 * initial authWithOAuth2 call in the synchronous click context.
 * See: https://github.com/pocketbase/pocketbase/discussions/2429#discussioncomment-5943061
 */
function loginWithMicrosoft() {
  pb.collection("users")
    .authWithOAuth2({ provider: "microsoft" })
    .then(() => {
      // PocketBase automatically updates pb.authStore on successful auth
      // The onChange callback will handle updating our Svelte store and setting up refresh
    })
    .catch((error) => {
      console.error("Microsoft login failed:", error);
    });
}

/**
 * AUTH ACTION: Logout and cleanup
 * Clears the refresh timer and PocketBase auth state
 * The onChange callback will update the Svelte store to null
 */
function logout() {
  // Clean up the refresh timer to prevent memory leaks
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }

  // Clear PocketBase auth state (token, user model, etc.)
  pb.authStore.clear();
  // The onChange callback will handle updating our Svelte store to null
}

// Export the auth store with all available methods
export const authStore = {
  // Utility methods
  tokenExpirationDate,

  // Svelte store interface
  subscribe,
  get: () => get(baseStore),

  // Auth actions
  loginWithMicrosoft,
  logout,

  // Core auth system methods
  refreshAuth,
  setupTokenRefresh,
  updateActivity,

  // Manual refresh for debugging/testing
  refresh: () => set(getCurrentAuthState()),
};
