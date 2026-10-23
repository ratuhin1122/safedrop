import { expect, it, vi } from "vitest";
import { downloadShare, downloadVariant } from "@/api/downloads";
import { setAccessToken } from "@/api/session";
import { fileId } from "@/api/fixtures.test-data";
import * as session from "@/api/session";

it("returns a stream for a protected variant instead of buffering the download", async () => {
  vi.stubEnv("VITE_API_URL", "/v1");
  setAccessToken("jwt");
  const response = new Response("encoded", { headers: { "Content-Type": "image/webp", "Content-Length": "7" } });
  const fetcher = vi.fn().mockResolvedValue(response);
  vi.stubGlobal("fetch", fetcher);
  const result = await downloadVariant(fileId, 375);
  expect(result.stream).toBe(response.body);
  expect(result.contentLength).toBe(7);
  expect(response.bodyUsed).toBe(false);
  expect(new Headers(fetcher.mock.calls[0][1].headers).get("Authorization")).toBe("Bearer jwt");
  await result.stream.cancel();
});

it("uses UTF-8 Basic auth for public shares and does not redirect a wrong password", async () => {
  vi.stubEnv("VITE_API_URL", "/v1");
  setAccessToken("private-jwt");
  const redirect = vi.spyOn(session, "handleUnauthorized").mockImplementation(() => {});
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "unauthorized", message: "Password incorrect." }), { status: 401 }));
  vi.stubGlobal("fetch", fetcher);
  await expect(downloadShare("A".repeat(22), "sécret")).rejects.toMatchObject({ status: 401, message: "Password incorrect." });
  expect(fetcher.mock.calls[0][0]).toBe(`/s/${"A".repeat(22)}`);
  const header = new Headers(fetcher.mock.calls[0][1].headers).get("Authorization");
  expect(header).toBe(`Basic ${btoa(String.fromCharCode(...new TextEncoder().encode(":sécret")))}`);
  expect(redirect).not.toHaveBeenCalled();
  redirect.mockRestore();
});
