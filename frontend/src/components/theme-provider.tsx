import { useEffect, useState, type ReactNode } from "react";
import { ThemeContext } from "@/hooks/use-theme";
import { applyTheme, savedTheme, systemTheme, themeKey } from "@/lib/theme";

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState(() => savedTheme() ?? systemTheme());

  useEffect(() => applyTheme(theme), [theme]);
  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const sync = () => setTheme(savedTheme() ?? systemTheme());
    media.addEventListener("change", sync);
    window.addEventListener("storage", sync);
    return () => {
      media.removeEventListener("change", sync);
      window.removeEventListener("storage", sync);
    };
  }, []);

  function toggleTheme() {
    const next = theme === "dark" ? "light" : "dark";
    try { localStorage.setItem(themeKey, next); } catch { /* Session theme still works. */ }
    setTheme(next);
  }

  return <ThemeContext.Provider value={{ theme, toggleTheme }}>{children}</ThemeContext.Provider>;
}
