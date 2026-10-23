import { z } from "zod";
import { getApiBase } from "@/lib/env";
import { errorSchema } from "@/api/schemas";
import { getAccessToken, handleUnauthorized } from "@/api/session";

export class ApiError extends Error {
  readonly retryAfterSeconds: number | null;
  constructor(public readonly code: string, message: string, public readonly status: number,
    public readonly retryAfter: string | null = null) {
    super(message);
    this.name = "ApiError";
    const seconds = retryAfter && /^\d+$/.test(retryAfter) ? Number(retryAfter)
      : retryAfter && /^[A-Za-z]/.test(retryAfter) ? Math.ceil((Date.parse(retryAfter) - Date.now()) / 1000) : NaN;
    this.retryAfterSeconds = Number.isFinite(seconds) ? Math.max(0, Number(seconds)) : null;
  }
}

export type RequestOptions = Omit<RequestInit, "headers"> & {
  token?: string | null;
  headers?: HeadersInit;
  auth?: boolean;
  scope?: "api" | "public";
};

export async function apiResponse(path: string, options: RequestOptions = {}): Promise<Response> {
  const pathname = path.split("?")[0];
  if (!path.startsWith("/") || path.startsWith("//") || /[\\#]/.test(path)
    || pathname.split("/").some((segment) => [".", ".."].includes(decodeURIComponent(segment)))) {
    throw new Error("API paths must be relative paths without traversal or fragments.");
  }
  const { token, auth = true, scope = "api", ...request } = options;
  const headers = new Headers(request.headers);
  headers.set("Accept", headers.get("Accept") ?? "application/json");
  const jwt = token === undefined ? getAccessToken() : token;
  if (auth && jwt) headers.set("Authorization", `Bearer ${jwt}`);
  // FormData must generate its own boundary, even if callers supplied JSON headers.
  if (request.body instanceof FormData) headers.delete("Content-Type");
  const base = scope === "public" ? getApiBase().replace(/\/v1$/, "") : getApiBase();
  let response: Response;
  try { response = await fetch(`${base}${path}`, { ...request, headers }); }
  catch (error) {
    if (request.signal?.aborted) throw error;
    throw new ApiError("network_error", "Could not reach SafeDrop. Check your connection and API address.", 0);
  }
  if (!response.ok) {
    const parsed = errorSchema.safeParse(await response.json().catch(() => undefined));
    const error = new ApiError(parsed.success ? parsed.data.code : "request_failed",
      parsed.success ? parsed.data.message : `SafeDrop returned HTTP ${response.status} without a valid error message.`,
      response.status, response.status === 429 ? response.headers.get("Retry-After") : null);
    // Credential failures and share-password challenges must remain on their own forms.
    if (response.status === 401 && auth) handleUnauthorized();
    throw error;
  }
  return response;
}

export async function apiRequest<T>(path: string, schema: z.ZodType<T>, options: RequestOptions = {}): Promise<T> {
  const response = await apiResponse(path, options);
  const payload: unknown = response.status === 204 ? undefined : await response.json().catch(() => undefined);
  const parsed = schema.safeParse(payload);
  if (!parsed.success) throw new ApiError("invalid_response", "SafeDrop returned an unexpected response format.", response.status);
  return parsed.data;
}

export function jsonBody(input: unknown): Pick<RequestOptions, "body" | "headers"> {
  return { body: JSON.stringify(input), headers: { "Content-Type": "application/json" } };
}
