let token: string | null = null;
let initialized = false;
const listeners = new Set<() => void>();
export const sessionKey = "safedrop-token";

export function setAccessToken(value: string | null) {
  initialized = true;
  token = value;
  try {
    if (value) sessionStorage.setItem(sessionKey, value);
    else sessionStorage.removeItem(sessionKey);
  } catch { /* Memory sessions still work when browser storage is blocked. */ }
  listeners.forEach((listener) => listener());
}

export function getAccessToken() {
  if (!initialized) {
    initialized = true;
    try { token = sessionStorage.getItem(sessionKey) || null; } catch { token = null; }
  }
  return token;
}

export function subscribeSession(listener: () => void) {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}

export function handleUnauthorized() {
  setAccessToken(null);
  if (window.location.pathname !== "/login") window.location.assign("/login");
}
