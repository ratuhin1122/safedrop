try {
  const saved = localStorage.getItem("safedrop-theme");
  const dark = saved === "dark" || (saved !== "light" && matchMedia("(prefers-color-scheme: dark)").matches);
  document.documentElement.classList.toggle("dark", dark);
  document.documentElement.style.colorScheme = dark ? "dark" : "light";
} catch { /* The theme provider follows system preference if storage is blocked. */ }
