export type Theme = "light" | "dark";
export const themeKey = "safedrop-theme";

export function savedTheme(): Theme | null {
  try {
    const value = localStorage.getItem(themeKey);
    return value === "light" || value === "dark" ? value : null;
  } catch { return null; }
}

export function systemTheme(): Theme {
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle("dark", theme === "dark");
  document.documentElement.style.colorScheme = theme;
  document.querySelector('meta[name="theme-color"]')?.setAttribute("content", theme === "dark" ? "#171e1b" : "#f6f5f0");
}
