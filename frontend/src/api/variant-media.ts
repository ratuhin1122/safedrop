import { apiResponse } from "@/api/client";
import { getApiBase } from "@/lib/env";

export async function variantBlob(url: string, signal?: AbortSignal) {
  const base = new URL(getApiBase() + "/", window.location.origin);
  const target = new URL(url, window.location.origin);
  if (target.origin !== base.origin || !target.pathname.startsWith(base.pathname)) {
    throw new Error("The preview URL is outside the configured API.");
  }
  const path = target.pathname.slice(base.pathname.length - 1) + target.search;
  const response = await apiResponse(path, { signal, headers: { Accept: "image/*" } });
  return response.blob();
}
