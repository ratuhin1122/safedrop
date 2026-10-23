import { createContext, useContext } from "react";
import type { Theme } from "@/lib/theme";

export const ThemeContext = createContext<{ theme: Theme; toggleTheme: () => void } | null>(null);

export function useTheme() {
  const value = useContext(ThemeContext);
  if (!value) throw new Error("useTheme must be used inside ThemeProvider.");
  return value;
}
