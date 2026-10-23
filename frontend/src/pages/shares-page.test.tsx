import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "vitest";
import { SharesPage } from "@/pages/shares-page";
import { ToastProvider } from "@/components/toast-provider";
import { listFiles } from "@/api/files";
import { listShares } from "@/api/shares";
import { fileFixture } from "@/api/fixtures.test-data";

vi.mock("@/api/files", () => ({ listFiles: vi.fn() }));
vi.mock("@/api/shares", () => ({ listShares: vi.fn() }));
it("discovers links across file pages using next_cursor", async () => {
  vi.mocked(listFiles).mockResolvedValueOnce({ items: [{ ...fileFixture, status: "ready" }], next_cursor: "next" })
    .mockResolvedValueOnce({ items: [{ ...fileFixture, id: "22222222-2222-4222-8222-222222222222", original_name: "second.png", status: "ready" }], next_cursor: "" });
  vi.mocked(listShares).mockImplementation(async (fileId) => ({ items: [{
    id: fileId, file_id: fileId, expires_at: "2099-01-01T00:00:00Z", max_downloads: null, download_count: 0, password_protected: false,
  }] }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  render(<QueryClientProvider client={client}><ToastProvider><MemoryRouter><SharesPage /></MemoryRouter></ToastProvider></QueryClientProvider>);
  expect(await screen.findByText("second.png")).toBeVisible();
  expect(listFiles).toHaveBeenNthCalledWith(2, { cursor: "next", limit: 24 }, expect.any(AbortSignal));
  expect(screen.getByText("image.jpg")).toBeVisible();
  client.clear();
});
