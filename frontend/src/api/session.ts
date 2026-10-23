let token: string | null = null;

// Memory-only: the authentication provider can restore a session explicitly.
export function setAccessToken(value: string | null) { token = value; }
export function getAccessToken() { return token; }

export function handleUnauthorized() {
  setAccessToken(null);
  if (window.location.pathname !== "/login") window.location.assign("/login");
}
