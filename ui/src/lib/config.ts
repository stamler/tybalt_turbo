// Auth configuration constants
export const AUTH_CONFIG = {
  // Refresh before expiry so users don't encounter expired-token saves.
  TOKEN_REFRESH_BUFFER_MINUTES: 5,

  // Browser/user events that tell the auth store the app is awake and can
  // opportunistically restart or refresh a session.
  SESSION_CHECK_EVENTS: ["mousedown", "keypress", "touchstart", "click"],
} as const;

// Convert minutes to milliseconds for easier use
export const AUTH_TIMEOUTS = {
  TOKEN_REFRESH_BUFFER_MS: AUTH_CONFIG.TOKEN_REFRESH_BUFFER_MINUTES * 60 * 1000,
} as const;
