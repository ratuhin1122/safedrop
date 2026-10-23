import { expect, it, vi } from "vitest";
import { z } from "zod";
import { apiRequest, ApiError } from "@/api/client";
import { setAccessToken } from "@/api/session";
import * as session from "@/api/session";

it("uses the configured API base, preserves its path, and validates responses", async () => {
  vi.stubEnv("VITE_API_URL", "/backend/v1/");
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ token: "test" }), { status: 200 }));
  vi.stubGlobal("fetch", fetcher);
  const result = await apiRequest("/auth/login", z.object({ token: z.string() }), { method: "POST" });
  expect(result.token).toBe("test");
  expect(fetcher.mock.calls[0][0]).toBe("/backend/v1/auth/login");
});

it("adds the current JWT and redirects a protected 401 while preserving its error", async () => {
  vi.stubEnv("VITE_API_URL", "/v1");
  setAccessToken("jwt-value");
  const redirect = vi.spyOn(session, "handleUnauthorized").mockImplementation(() => setAccessToken(null));
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "unauthorized", message: "Token expired." }), { status: 401 }));
  vi.stubGlobal("fetch", fetcher);
  await expect(apiRequest("/files", z.unknown())).rejects.toMatchObject({ status: 401, message: "Token expired." });
  expect(new Headers(fetcher.mock.calls[0][1].headers).get("Authorization")).toBe("Bearer jwt-value");
  expect(redirect).toHaveBeenCalledOnce();
  expect(session.getAccessToken()).toBeNull();
  redirect.mockRestore();
});

it("does not redirect or send the JWT for a public password challenge", async () => {
  vi.stubEnv("VITE_API_URL", "/v1");
  setAccessToken("private-jwt");
  const redirect = vi.spyOn(session, "handleUnauthorized").mockImplementation(() => {});
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "unauthorized", message: "Wrong password." }), { status: 401 }));
  vi.stubGlobal("fetch", fetcher);
  await expect(apiRequest("/s/token", z.unknown(), { auth: false, scope: "public" })).rejects.toMatchObject({ message: "Wrong password." });
  expect(new Headers(fetcher.mock.calls[0][1].headers).has("Authorization")).toBe(false);
  expect(redirect).not.toHaveBeenCalled();
  redirect.mockRestore();
});

it("validates successful payloads and does not follow paths outside the API", async () => {
  vi.stubEnv("VITE_API_URL", "/v1");
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ token: 12 }), { status: 200 }));
  vi.stubGlobal("fetch", fetcher);
  await expect(apiRequest("/auth/login", z.object({ token: z.string() }))).rejects.toMatchObject({ code: "invalid_response" });
  await expect(apiRequest("//other-host", z.unknown())).rejects.toThrow("relative paths");
  await expect(apiRequest("/%2e%2e/private", z.unknown())).rejects.toThrow("relative paths");
  expect(fetcher).toHaveBeenCalledOnce();
});

it("preserves the API's specific error message and retry header", async () => {
  vi.stubEnv("VITE_API_URL", "/v1");
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
    JSON.stringify({ code: "rate_limited", message: "Try again in one minute." }),
    { status: 429, headers: { "Retry-After": "60" } },
  )));
  await expect(apiRequest("/files", z.unknown())).rejects.toMatchObject({
    code: "rate_limited", message: "Try again in one minute.", status: 429, retryAfter: "60",
    retryAfterSeconds: 60,
  });
});

it("handles date-valued and malformed Retry-After headers", () => {
  expect(new ApiError("rate_limited", "Wait", 429, "Thu, 01 Jan 2099 00:00:00 GMT").retryAfterSeconds).toBeGreaterThan(0);
  expect(new ApiError("rate_limited", "Wait", 429, "-1").retryAfterSeconds).toBeNull();
  expect(new ApiError("rate_limited", "Wait", 429).retryAfterSeconds).toBeNull();
});
