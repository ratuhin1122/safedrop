import { apiRequest } from "@/api/client";
import { healthSchema, usageSchema } from "@/api/schemas";

export function getUsage(signal?: AbortSignal) {
  return apiRequest("/me/usage", usageSchema, { signal });
}

export function getHealth(signal?: AbortSignal) {
  return apiRequest("/healthz", healthSchema, { signal, auth: false, scope: "public" });
}
