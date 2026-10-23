import { useEffect, useMemo, useSyncExternalStore, type ReactNode } from "react";
import { login, register } from "@/api/auth";
import { getAccessToken, setAccessToken, subscribeSession } from "@/api/session";
import { AuthContext, type AuthState } from "@/hooks/use-auth";

export function AuthProvider({ children, onSessionChange }: { children: ReactNode; onSessionChange?: () => void }) {
  const token = useSyncExternalStore(subscribeSession, getAccessToken, () => null);
  useEffect(() => { onSessionChange?.(); }, [token, onSessionChange]);
  const value = useMemo<AuthState>(() => ({
    token, isAuthenticated: Boolean(token),
    signIn: async (input, signal) => { await login(input, signal); },
    signUp: async (input, signal) => { await register(input, signal); },
    signOut: () => setAccessToken(null),
  }), [token]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
