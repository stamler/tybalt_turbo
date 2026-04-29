import { writable, get } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { getTokenPayload, isTokenExpired } from "pocketbase";
import type { AuthModel, SendOptions } from "pocketbase";
import { AUTH_TIMEOUTS } from "$lib/config";

/**
 * Browser auth coordinator
 * ========================
 *
 * PocketBase owns the durable browser auth state in `pb.authStore`:
 * - token: the JWT sent to the backend
 * - record: the authenticated user record
 *
 * This module wraps that state in a Svelte store and adds the app-level behavior
 * that PocketBase does not provide for us out of the box:
 *
 * 1. Expiry-aware refresh scheduling with no app-specific idle timeout
 *    We refresh shortly before the JWT expires instead of on a fixed interval.
 *    The app does not stop refreshing just because the user was inactive. As
 *    long as PocketBase can refresh the OAuth-created auth session, the user
 *    stays signed in.
 *
 * 2. Wake/resume checks
 *    Browser timers can be delayed while a tab is backgrounded or a computer is
 *    asleep. Focus, visibility, online, and user-interaction events give us a
 *    chance to restart the refresh timer or refresh immediately when the user
 *    returns.
 *
 * 3. Request-time freshness guard
 *    Before protected PocketBase requests, we check whether the token is near
 *    expiry. If it is, we refresh first and update the outgoing Authorization
 *    header so the original request does not leave with a stale token.
 *
 * 4. Narrow 401 probing
 *    This backend returns HTTP 401 for some normal app-level authorization
 *    failures, so a 401 response must not automatically mean "log the user out".
 *    Instead, after a non-auth 401, we force an auth refresh. A failed refresh
 *    proves the session is no longer valid and clears auth; a successful refresh
 *    leaves the original 401 for the calling page to handle.
 *
 * 5. Session-expired notification
 *    When auth is cleared because the session truly expired, we dispatch a
 *    browser event. `hooks.client.ts` turns that into a global UI error, while
 *    `+layout.svelte` reacts to the Svelte auth store becoming invalid and
 *    redirects to `/login`.
 *
 * User-visible behavior
 * ---------------------
 *
 * - A user should stay signed in whether they are actively working or have left
 *   the app idle, as long as PocketBase can refresh the session. The app relies
 *   on PocketBase auth validity and the original Microsoft OAuth login rather
 *   than an app-specific inactivity timeout.
 *
 * - `authRefresh` refreshes the PocketBase JWT; it does not silently rerun the
 *   Microsoft OAuth popup/redirect. If PocketBase refresh eventually fails,
 *   auth is cleared, a "session expired" message is emitted, and the layout
 *   sends them to `/login?redirect=...`. If Microsoft still has a valid browser
 *   session, logging back in is usually just another quick OAuth pass.
 *
 * - A user may be redirected later than the exact instant a refresh would have
 *   failed if the tab is asleep or background timers are throttled. In that case
 *   the check happens when the app wakes up, receives activity/focus/online
 *   events, or attempts a protected request.
 *
 * - A user who is authenticated but lacks permission for a specific action
 *   should not be logged out. Some app routes return 401 for authorization
 *   failures. We probe the session with authRefresh, but if refresh succeeds the
 *   original 401 remains an action-level error for the page to display.
 *
 * - Manual logout is quiet. It clears auth and returns the app to login through
 *   the normal route guard, but it does not show the session-expired message.
 */
export const AUTH_SESSION_EXPIRED_EVENT = "tybalt:session-expired";

// The shape exposed to Svelte components. `null` means unauthenticated.
interface AuthState {
  isValid: boolean;
  model: AuthModel;
  token: string;
}

type EnsureFreshAuthOptions = {
  /**
   * When false, `ensureFreshAuth` only refreshes inside the configured
   * pre-expiry buffer. When true, it calls PocketBase authRefresh even if the
   * JWT still has plenty of time remaining. Forced refresh is useful as a
   * server-side validity probe after a 401 or during startup.
   */
  force?: boolean;
};

type TokenTimingMilliseconds = {
  issuedAt: number;
  expiresAt: number;
  lifetime: number;
};

const SESSION_EXPIRED_MESSAGE = "Your session expired. Sign in again to continue.";
const MIN_TOKEN_REFRESH_DELAY_MS = 10 * 1000;
const MIN_REFRESHABLE_TOKEN_LIFETIME_MS = 30 * 1000;
const SHORT_TOKEN_REFRESH_FRACTION = 0.8;
const baseStore = writable<AuthState | null>(null);
const { subscribe, set } = baseStore;

// Schedules the next proactive refresh shortly before the current token expires.
let refreshTimer: ReturnType<typeof setTimeout> | null = null;

// Shared in-flight refresh. This prevents request bursts from issuing multiple
// simultaneous authRefresh calls when several requests all notice near-expiry
// auth at the same time.
let refreshPromise: Promise<boolean> | null = null;

// Prevents one expired session from spamming the same global error while several
// requests/timers all notice the invalid auth state.
let sessionExpiredNotified = false;

// PocketBase only has one beforeSend/afterSend slot. Guard setup is idempotent
// so hooks.client.ts can call it confidently during client startup.
let requestAuthGuardSetup = false;

/**
 * Reads the JWT `exp` claim from the PocketBase auth token.
 *
 * This is intentionally based on the token payload, not a server round-trip.
 * It lets the browser schedule refresh/expiry work cheaply. Server-side
 * validity is checked only by `authRefresh`.
 */
function tokenExpirationDate(): Date | null {
  const exp = getTokenPayload(pb.authStore.token)?.exp;
  if (!exp) return null;

  const expirationDate = new Date(exp * 1000);
  return isNaN(expirationDate.getTime()) ? null : expirationDate;
}

function tokenTimingMilliseconds(): TokenTimingMilliseconds | null {
  const payload = getTokenPayload(pb.authStore.token);
  const exp = payload?.exp;
  const iat = payload?.iat;
  if (typeof exp !== "number" || typeof iat !== "number") return null;

  const issuedAt = iat * 1000;
  const expiresAt = exp * 1000;
  const lifetime = expiresAt - issuedAt;
  return lifetime > 0 ? { issuedAt, expiresAt, lifetime } : null;
}

/**
 * Returns true when the token payload is structurally parseable and not expired.
 *
 * Important: this is a local JWT check only. PocketBase also verifies tokens
 * against server-side material such as the auth record's tokenKey and the auth
 * collection secret. A locally valid token can still fail `authRefresh` if the
 * user was deleted, their tokenKey changed, or collection secrets changed.
 */
function checkTokenExpiration(token: string): boolean {
  return !isTokenExpired(token);
}

function millisecondsUntilTokenExpiration(): number | null {
  const expiration = tokenExpirationDate();
  if (!expiration) return null;

  return expiration.getTime() - Date.now();
}

/**
 * Schedules auth cleanup when a token is intentionally not worth refreshing.
 *
 * This is only for pathological/dev token lifetimes below
 * MIN_REFRESHABLE_TOKEN_LIFETIME_MS. Normal sessions should refresh before they
 * reach expiry; very short sessions are allowed to expire so the app avoids
 * hammering authRefresh forever.
 */
function scheduleSessionExpiration() {
  clearRefreshTimer();

  const millisecondsUntilExpiration = millisecondsUntilTokenExpiration();
  if (millisecondsUntilExpiration === null || millisecondsUntilExpiration <= 0) {
    expireSession();
    return;
  }

  refreshTimer = setTimeout(expireSession, millisecondsUntilExpiration + 1000);
}

/**
 * Calculates when the next proactive refresh should run.
 *
 * The normal schedule is:
 *
 *   token expiry - TOKEN_REFRESH_BUFFER_MS
 *
 * The minimum delay guard prevents a tight refresh loop when a development or
 * misconfigured PocketBase token lifetime is shorter than the frontend refresh
 * buffer. For example, a 60 second token with a 5 minute buffer would otherwise
 * schedule refresh at 0ms, each refresh would return another 60 second token,
 * and the app would continuously call authRefresh.
 *
 * When the configured buffer is larger than the token's whole lifetime, we
 * refresh once the token has used most of its lifetime instead. The refresh
 * point is calculated from `iat`, not from the moving "remaining time", so a 60
 * second token refreshes around 48 seconds after issue rather than before every
 * protected request.
 */
function millisecondsUntilNextRefresh(): number | null {
  const millisecondsUntilExpiration = millisecondsUntilTokenExpiration();
  if (millisecondsUntilExpiration === null || millisecondsUntilExpiration <= 0) {
    return null;
  }

  const timing = tokenTimingMilliseconds();
  if (!timing) {
    // PocketBase JWTs include `iat`; this fallback keeps exp-only tokens usable
    // instead of treating a missing issued-at claim as an immediate logout.
    return millisecondsUntilExpiration - AUTH_TIMEOUTS.TOKEN_REFRESH_BUFFER_MS;
  }

  if (timing.lifetime < MIN_REFRESHABLE_TOKEN_LIFETIME_MS) {
    return null;
  }

  const normalRefreshAt = timing.expiresAt - AUTH_TIMEOUTS.TOKEN_REFRESH_BUFFER_MS;
  if (normalRefreshAt > timing.issuedAt) {
    return normalRefreshAt - Date.now();
  }

  const shortTokenRefreshDelay = Math.max(
    MIN_TOKEN_REFRESH_DELAY_MS,
    Math.floor(timing.lifetime * SHORT_TOKEN_REFRESH_FRACTION),
  );
  const shortTokenRefreshAt = timing.issuedAt + shortTokenRefreshDelay;
  return shortTokenRefreshAt - Date.now();
}

function clearRefreshTimer() {
  if (refreshTimer) {
    clearTimeout(refreshTimer);
    refreshTimer = null;
  }
}

/**
 * Broadcasts a user-visible session-expired message.
 *
 * The auth store does not import `globalStore` directly to avoid turning this
 * low-level auth module into a UI store dependency. Instead, hooks.client.ts
 * listens for this event and decides how to display it.
 */
function notifySessionExpired(message = SESSION_EXPIRED_MESSAGE) {
  if (sessionExpiredNotified || typeof window === "undefined") return;

  sessionExpiredNotified = true;
  window.dispatchEvent(new CustomEvent(AUTH_SESSION_EXPIRED_EVENT, { detail: { message } }));
}

/**
 * Clears all frontend auth state after we have determined the session is dead.
 *
 * Calling `pb.authStore.clear()` triggers PocketBase's auth onChange callback in
 * hooks.client.ts, which refreshes this Svelte store to `null`. The root layout
 * then observes `$authStore?.isValid` becoming false and redirects to login.
 *
 * User impact: this is the point where the current screen stops being an
 * authenticated app screen. The next layout reaction sends the user to login,
 * preserving the current path in the redirect query parameter.
 */
function expireSession(message = SESSION_EXPIRED_MESSAGE) {
  clearRefreshTimer();
  refreshPromise = null;
  notifySessionExpired(message);

  if (pb.authStore.token || pb.authStore.record) {
    pb.authStore.clear();
  } else {
    set(null);
  }
}

/**
 * Identifies endpoints that are part of the auth mechanism itself.
 *
 * The request guard must not try to refresh before calling auth-refresh, or
 * force another refresh in response to auth-refresh failing. That would create
 * recursion and hide the real failure.
 */
function isAuthEndpoint(url: string): boolean {
  let pathname = url;
  try {
    pathname = new URL(url, typeof window === "undefined" ? undefined : window.location.origin)
      .pathname;
  } catch {
    // Keep the original string; the suffix checks below still work for SDK paths.
  }

  return [
    "/api/collections/users/auth-refresh",
    "/api/collections/users/auth-with-oauth2",
    "/api/collections/users/auth-methods",
  ].some((path) => pathname.endsWith(path));
}

/**
 * PocketBase adds an Authorization header during request initialization. If a
 * beforeSend refresh replaces the token, the request's headers may still hold
 * the old token. This helper overwrites the outgoing header with the fresh one.
 */
function setAuthorizationHeader(options: SendOptions, token: string) {
  options.headers = {
    ...(options.headers ?? {}),
    Authorization: token,
  };
}

/**
 * Removes any Authorization header, regardless of casing.
 *
 * If a refresh attempt fails in beforeSend, `expireSession()` clears
 * `pb.authStore`, but the current request options may still contain the stale
 * Authorization header that PocketBase inserted before beforeSend ran. Removing
 * it prevents one last request from leaving with known-bad credentials.
 */
function removeAuthorizationHeader(options: SendOptions) {
  const headers = { ...(options.headers ?? {}) };
  for (const key of Object.keys(headers)) {
    if (key.toLowerCase() === "authorization") {
      delete headers[key];
    }
  }
  options.headers = headers;
}

/**
 * Ensures the current PocketBase auth can be used safely.
 *
 * Return value:
 * - true: there is a valid auth record/token and no logout is needed
 * - false: there is no usable auth; when auth existed but was invalid, this
 *   function has already cleared the session
 *
 * Non-forced calls are cheap most of the time: they only inspect the token's
 * local expiry and return true until the shared refresh schedule says a refresh
 * is due. For normal tokens that means "before the configured pre-expiry buffer
 * begins"; for short development tokens it means "before the token has used
 * most of its lifetime." To the user, this is the common path where nothing
 * visible happens and their work continues normally.
 *
 * Forced calls always hit PocketBase's authRefresh endpoint. That is our
 * canonical server-side session validity check. It verifies the JWT against the
 * current auth record/tokenKey/collection secret and returns a new token when
 * everything is still valid.
 *
 * User impact:
 * - successful refresh: the user stays on the same screen and can keep working
 * - failed refresh: the session is cleared, the user sees the expired-session
 *   message, and the layout redirects them to login
 */
async function ensureFreshAuth(options: EnsureFreshAuthOptions = {}): Promise<boolean> {
  const { force = false } = options;
  const token = pb.authStore.token;

  if (!token || !pb.authStore.record) return false;

  if (!checkTokenExpiration(token)) {
    expireSession();
    return false;
  }

  if (!force) {
    const millisecondsUntilRefresh = millisecondsUntilNextRefresh();
    if (millisecondsUntilRefresh === null) {
      const timing = tokenTimingMilliseconds();
      if (timing && timing.lifetime < MIN_REFRESHABLE_TOKEN_LIFETIME_MS) {
        scheduleSessionExpiration();
        return true;
      }

      expireSession();
      return false;
    }

    if (millisecondsUntilRefresh > 0) {
      return true;
    }
  }

  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    try {
      // `requestKey` avoids default auto-cancellation collisions while several
      // callers await the same refreshPromise.
      await pb.collection("users").authRefresh({ requestKey: "auth-refresh" });
      sessionExpiredNotified = false;
      return true;
    } catch (error) {
      console.warn("Token refresh failed:", error);
      expireSession();
      return false;
    }
  })().finally(() => {
    refreshPromise = null;
  });

  return refreshPromise;
}

/**
 * Public forced refresh helper used during startup and by a few call sites that
 * want to explicitly validate the browser's existing PocketBase auth state.
 */
async function refreshAuth(): Promise<boolean> {
  return ensureFreshAuth({ force: true });
}

/**
 * Runs when the proactive refresh timer fires.
 *
 * There is deliberately no inactivity check here. The product model is "keep
 * trying to refresh for as long as PocketBase auth permits it." If
 * refresh succeeds, PocketBase updates `pb.authStore` and hooks.client.ts
 * schedules the next timer from the new token expiry. If refresh fails,
 * ensureFreshAuth clears the session and the layout sends the user to login.
 *
 * User impact: leaving the app open overnight should not by itself end the
 * session. The session ends when the upstream auth chain can no longer refresh.
 */
async function handleScheduledRefresh() {
  refreshTimer = null;

  if (!pb.authStore.token || !pb.authStore.record) return;
  const success = await ensureFreshAuth({ force: true });
  if (success && !refreshTimer) setupTokenRefresh();
}

/**
 * Creates the next expiry-aware auth timer.
 *
 * This is called after login, startup refresh, PocketBase auth changes, and
 * wake/activity events that restart the system. It schedules refresh at:
 *
 *   token expiry - TOKEN_REFRESH_BUFFER_MS
 *
 * If the token lifetime is shorter than the configured buffer, a minimum delay
 * is used to avoid an immediate authRefresh loop.
 *
 * If the token's whole lifetime is below MIN_REFRESHABLE_TOKEN_LIFETIME_MS, the
 * app gives up on proactive refresh and schedules normal expiry cleanup. That
 * keeps absurdly short development token lifetimes from turning into a refresh
 * storm.
 */
function setupTokenRefresh() {
  clearRefreshTimer();

  if (!pb.authStore.token || !pb.authStore.record) return;

  sessionExpiredNotified = false;

  const millisecondsUntilRefresh = millisecondsUntilNextRefresh();
  if (millisecondsUntilRefresh === null) {
    scheduleSessionExpiration();
    return;
  }

  refreshTimer = setTimeout(handleScheduledRefresh, millisecondsUntilRefresh);
}

/**
 * Opportunistically keeps auth healthy when the app wakes or receives input.
 *
 * hooks.client.ts wires this to mouse, keyboard, touch, visibility, focus, and
 * online events. These events do two things:
 * - restart the refresh timer if a browser sleep/background period left none
 * - trigger an immediate refresh only when the token is close enough to expiry
 *   that waiting for another timer would be risky
 *
 * User impact: returning to a sleeping tab, focusing the window, or coming back
 * online gives the app a chance to refresh before the next save/load action.
 */
function updateActivity() {
  if (!pb.authStore.token || !pb.authStore.record) return;

  if (!checkTokenExpiration(pb.authStore.token)) {
    expireSession();
    return;
  }

  if (!refreshTimer) {
    setupTokenRefresh();
  }

  const millisecondsUntilExpiration = millisecondsUntilTokenExpiration();
  if (
    millisecondsUntilExpiration !== null &&
    millisecondsUntilExpiration <= MIN_TOKEN_REFRESH_DELAY_MS
  ) {
    void ensureFreshAuth();
  }
}

/**
 * Converts PocketBase's authStore into our Svelte AuthState.
 *
 * This is deliberately validating, not just copying. If the browser still has a
 * token but it is locally expired, this clears auth and returns null so the UI
 * reacts as unauthenticated.
 */
function getCurrentAuthState(): AuthState | null {
  if (!pb.authStore.token || !pb.authStore.record) {
    return null;
  }

  if (!checkTokenExpiration(pb.authStore.token)) {
    expireSession();
    return null;
  }

  return {
    isValid: true,
    model: pb.authStore.record,
    token: pb.authStore.token,
  };
}

/**
 * Installs PocketBase request hooks for the whole app.
 *
 * beforeSend:
 *   Runs before every SDK request. For non-auth endpoints with a current user,
 *   it refreshes near-expiry tokens before the request is sent, then rewrites
 *   the Authorization header with the newest token.
 *
 * afterSend:
 *   Runs after every SDK response. A non-auth 401 is ambiguous in this app:
 *   it might mean "your session is invalid" or it might be a custom backend
 *   route saying "this authenticated user is not allowed to do that".
 *
 *   To distinguish those cases, we force authRefresh as a probe:
 *   - refresh fails: the session is invalid and ensureFreshAuth clears auth
 *   - refresh succeeds: the session is fine; return the original 401 data so
 *     the caller can show the normal authorization error
 *
 * User impact:
 * - if the 401 was caused by a dead session, the user is returned to login
 * - if the 401 was caused by missing permission, the user remains signed in and
 *   the relevant page/action can show its usual error message
 *
 * We intentionally do not retry the original request here. A successful probe
 * only answers "is this session still valid?", not "should this action now be
 * allowed?" Retrying would risk duplicate writes and would hide app-level auth
 * errors from their natural call sites.
 */
function setupRequestAuthGuard() {
  if (requestAuthGuardSetup) return;
  requestAuthGuardSetup = true;

  pb.beforeSend = async (url, options) => {
    if (!isAuthEndpoint(url) && pb.authStore.token && pb.authStore.record) {
      const isFresh = await ensureFreshAuth();
      if (isFresh && pb.authStore.token) {
        setAuthorizationHeader(options, pb.authStore.token);
      } else {
        removeAuthorizationHeader(options);
      }
    }

    return { url, options };
  };

  pb.afterSend = async (response, data) => {
    if (
      response.status === 401 &&
      !isAuthEndpoint(response.url) &&
      pb.authStore.token &&
      pb.authStore.record
    ) {
      await ensureFreshAuth({ force: true });
    }

    return data;
  };
}

/**
 * Starts the Microsoft OAuth flow.
 *
 * This function intentionally uses promise chaining rather than being marked
 * `async`. Browser popup blockers are sensitive to whether window.open happens
 * in the direct synchronous call stack of a user click; PocketBase's OAuth flow
 * relies on that behavior.
 */
function loginWithMicrosoft() {
  sessionExpiredNotified = false;
  pb.collection("users")
    .authWithOAuth2({ provider: "microsoft" })
    .then(() => {
      // PocketBase updates pb.authStore; hooks.client.ts syncs the Svelte store.
    })
    .catch((error) => {
      console.error("Microsoft login failed:", error);
    });
}

/**
 * User-initiated logout.
 *
 * This is not treated as an expired session: it clears timers and auth state
 * without dispatching the session-expired notification.
 */
function logout() {
  clearRefreshTimer();
  refreshPromise = null;
  sessionExpiredNotified = false;
  pb.authStore.clear();
}

/**
 * Svelte store facade exported to the rest of the app.
 *
 * Most components should only subscribe or call `loginWithMicrosoft`/`logout`.
 * hooks.client.ts owns the lifecycle methods (`setupRequestAuthGuard`,
 * `setupTokenRefresh`, `updateActivity`, and `refresh`) because it is the one
 * browser-only module that runs once at app startup.
 */
export const authStore = {
  tokenExpirationDate,

  subscribe,
  get: () => get(baseStore),

  loginWithMicrosoft,
  logout,

  ensureFreshAuth,
  refreshAuth,
  setupRequestAuthGuard,
  setupTokenRefresh,
  updateActivity,

  refresh: () => set(getCurrentAuthState()),
};
