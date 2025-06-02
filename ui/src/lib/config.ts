// Auth configuration constants
export const AUTH_CONFIG = {
  // Inactivity timeout - how long user can be idle before stopping token refresh
  INACTIVITY_TIMEOUT_MINUTES: 30,
  
  // Token refresh interval - how often to refresh tokens for active users
  TOKEN_REFRESH_INTERVAL_MINUTES: 45,
  
  // Events that count as user activity
  ACTIVITY_EVENTS: ['mousedown', 'mousemove', 'keypress', 'scroll', 'touchstart', 'click']
} as const;

// Convert minutes to milliseconds for easier use
export const AUTH_TIMEOUTS = {
  INACTIVITY_TIMEOUT_MS: AUTH_CONFIG.INACTIVITY_TIMEOUT_MINUTES * 60 * 1000,
  TOKEN_REFRESH_INTERVAL_MS: AUTH_CONFIG.TOKEN_REFRESH_INTERVAL_MINUTES * 60 * 1000
} as const; 