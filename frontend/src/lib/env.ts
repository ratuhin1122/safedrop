import { z } from "zod";

const apiBase = z.string().trim().min(1, "Set VITE_API_URL in frontend/.env.").refine((value) => {
  if (value.startsWith("/") && !value.startsWith("//")) return !/[?#\\]/.test(value);
  try {
    const url = new URL(value);
    return ["http:", "https:"].includes(url.protocol) && !url.username && !url.password && !url.search && !url.hash;
  } catch { return false; }
}, "VITE_API_URL must be an HTTP(S) API base URL or a same-origin path.");

export function getApiBase(): string {
  return apiBase.parse(import.meta.env.VITE_API_URL ?? "").replace(/\/+$/, "");
}

export const turnstileSiteKey = import.meta.env.VITE_TURNSTILE_SITE_KEY?.trim() ?? "";
