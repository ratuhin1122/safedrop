import { expect, it, vi } from "vitest";
import { z } from "zod";
import { apiRequest } from "@/api/client";

it("uses the configured API base, preserves its path, and validates responses", async () => {
  vi.stubEnv("VITE_API_URL", "/backend/v1/");
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ token: "test" }), { status: 200 }));
  vi.stubGlobal("fetch", fetcher);
  const result = await apiRequest("/auth/login", z.object({ token: z.string() }), { method: "POST" });
  expect(result.token).toBe("test");
  expect(fetcher.mock.calls[0][0]).toBe("/backend/v1/auth/login");
});

it("preserves the API's specific error message and retry header", async () => {
  vi.stubEnv("VITE_API_URL", "/v1");
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
    JSON.stringify({ code: "rate_limited", message: "Try again in one minute." }),
    { status: 429, headers: { "Retry-After": "60" } },
  )));
  await expect(apiRequest("/files", z.unknown())).rejects.toMatchObject({
    code: "rate_limited", message: "Try again in one minute.", status: 429, retryAfter: "60",
  });
});
