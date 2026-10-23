import { render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FileThumbnail } from "@/components/file-thumbnail";
import { fileFixture } from "@/api/fixtures.test-data";
import { setAccessToken } from "@/api/session";

it("loads the w375 WebP with authentication and revokes its object URL", async () => {
  vi.stubEnv("VITE_API_URL", "/v1");
  setAccessToken("thumbnail-jwt");
  const fetcher = vi.fn().mockResolvedValue(new Response(new Blob(["webp"], { type: "image/webp" })));
  vi.stubGlobal("fetch", fetcher);
  const create = vi.fn(() => "blob:thumbnail");
  const revoke = vi.fn();
  const originalCreate = URL.createObjectURL;
  const originalRevoke = URL.revokeObjectURL;
  URL.createObjectURL = create;
  URL.revokeObjectURL = revoke;
  try {
    const { unmount } = render(<FileThumbnail file={{ ...fileFixture, status: "ready" }} />);
    await waitFor(() => expect(screen.getByAltText("")).toHaveAttribute("src", "blob:thumbnail"));
    expect(fetcher.mock.calls[0][0]).toBe(fileFixture.variants.w375.url);
    expect(fetcher.mock.calls[0][1].headers.get("Authorization")).toBe("Bearer thumbnail-jwt");
    unmount();
    expect(revoke).toHaveBeenCalledWith("blob:thumbnail");
  } finally {
    URL.createObjectURL = originalCreate;
    URL.revokeObjectURL = originalRevoke;
  }
});
it("never sends credentials to an external metadata URL", () => {
  vi.stubEnv("VITE_API_URL", "/v1");
  const fetcher = vi.fn();
  vi.stubGlobal("fetch", fetcher);
  render(<FileThumbnail file={{ ...fileFixture, status: "ready",
    variants: { ...fileFixture.variants, w375: { ...fileFixture.variants.w375, url: "https://external.example/image" } } }} />);
  expect(fetcher).not.toHaveBeenCalled();
});
