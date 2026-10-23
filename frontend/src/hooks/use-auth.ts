import { createContext, useContext } from "react";
import type { Credentials, RegisterInput } from "@/api/schemas";

export type AuthState = {
  token: string | null;
  isAuthenticated: boolean;
  signIn: (input: Credentials, signal?: AbortSignal) => Promise<void>;
  signUp: (input: RegisterInput, signal?: AbortSignal) => Promise<void>;
  signOut: () => void;
};
export const AuthContext = createContext<AuthState | null>(null);

export function useAuth() {
  const auth = useContext(AuthContext);
  if (!auth) throw new Error("useAuth must be used inside AuthProvider.");
  return auth;
}
