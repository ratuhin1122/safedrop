import { expect, it, vi } from "vitest";
import { getPublicShare } from "@/api/public-share";
import { setAccessToken } from "@/api/session";

it("requests metadata without a JWT and sends a password only in Basic authentication", async () => {
  vi.stubEnv("VITE_API_URL", "https://api.example/v1");
  setAccessToken("private-account-token");
  const fetcher = vi.fn().mockImplementation(async () => new Response(JSON.stringify({
    name: "image.webp", size: 10, mime: "image/webp", expires_at: "2099-01-01T00:00:00Z",
  }), { status: 200 }));
  vi.stubGlobal("fetch", fetcher);
  const token = "MDEyMzQ1Njc4OWFiY2RlZg";
  await getPublicShare(token);
  expect(fetcher.mock.calls[0][0]).toBe("https://api.example/s/" + token + "?info=1");
  expect(fetcher.mock.calls[0][1].headers.has("Authorization")).toBe(false);
  await getPublicShare(token, "secret");
  expect(fetcher.mock.calls[1][1].headers.get("Authorization")).toBe("Basic " + btoa(":secret"));
  expect(fetcher.mock.calls[1][0]).not.toContain("secret");
});
