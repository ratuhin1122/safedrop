import { z } from "zod";
import { getApiBase } from "@/lib/env";

const errorSchema = z.object({ code: z.string(), message: z.string() });

export class ApiError extends Error {
  constructor(public readonly code: string, message: string, public readonly status: number,
    public readonly retryAfter: string | null = null) {
    super(message);
    this.name = "ApiError";
  }
}

type Options = Omit<RequestInit, "headers"> & { token?: string; headers?: HeadersInit };

// Callers provide a response schema; payloads never become unchecked `any`.
export async function apiRequest<T>(path: string, schema: z.ZodType<T>, options: Options = {}): Promise<T> {
  if (!path.startsWith("/") || path.startsWith("//")) throw new Error("API paths must start with one slash.");
  const { token, ...request } = options;
  const headers = new Headers(request.headers);
  headers.set("Accept", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);
  // FormData sets its own multipart boundary; do not set Content-Type here.
  let response: Response;
  const url = `${getApiBase()}${path}`;
  try { response = await fetch(url, { ...request, headers }); }
  catch (error) {
    if (request.signal?.aborted) throw error;
    throw new ApiError("network_error", "Could not reach SafeDrop. Check your connection and API address.", 0);
  }
  const payload: unknown = response.status === 204 ? undefined : await response.json().catch(() => undefined);
  if (!response.ok) {
    const parsed = errorSchema.safeParse(payload);
    throw new ApiError(parsed.success ? parsed.data.code : "request_failed",
      parsed.success ? parsed.data.message : `SafeDrop returned HTTP ${response.status} without a valid error message.`,
      response.status, response.headers.get("Retry-After"));
  }
  const parsed = schema.safeParse(payload);
  if (!parsed.success) throw new ApiError("invalid_response", "SafeDrop returned an unexpected response format.", response.status);
  return parsed.data;
}
