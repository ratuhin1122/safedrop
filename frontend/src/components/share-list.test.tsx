import { fireEvent, render, screen, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, expect, it, vi } from "vitest";
import { ShareList } from "@/components/share-list";
import { ToastProvider } from "@/components/toast-provider";
import { deleteShare, listShares } from "@/api/shares";
import { rememberShare } from "@/hooks/use-share-history";
import { fileId } from "@/api/fixtures.test-data";
import type { ShareItem } from "@/api/schemas";

vi.mock("@/api/shares", () => ({ deleteShare: vi.fn(), listShares: vi.fn() }));
const share: ShareItem = {
  id: "22222222-2222-4222-8222-222222222222", file_id: fileId,
  expires_at: "2099-01-01T00:00:00Z", max_downloads: 3, download_count: 1, password_protected: false,
};
beforeEach(() => {
  vi.clearAllMocks();
  HTMLDialogElement.prototype.showModal = function () { this.setAttribute("open", ""); };
  HTMLDialogElement.prototype.close = function () { this.removeAttribute("open"); };
});
function setup(shares = [share], client = new QueryClient()) {
  render(<QueryClientProvider client={client}><ToastProvider><ShareList shares={shares} /></ToastProvider></QueryClientProvider>);
  return client;
}
it("revokes with confirmation and updates the existing row without a reload", async () => {
  vi.mocked(deleteShare).mockResolvedValue(undefined);
  setup();
  expect(screen.getByText("Active")).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "Revoke" }));
  expect(deleteShare).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Confirm revoke" }));
  expect(await screen.findByText("Revoked")).toBeVisible();
  expect(deleteShare).toHaveBeenCalledWith(share.id);
  expect(screen.getByRole("button", { name: "Revoke" })).toBeDisabled();
});
it("keeps the link active and reports revoke errors", async () => {
  vi.mocked(deleteShare).mockRejectedValue(new Error("Permission denied"));
  setup();
  fireEvent.click(screen.getByRole("button", { name: "Revoke" }));
  fireEvent.click(screen.getByRole("button", { name: "Confirm revoke" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Permission denied");
  expect(screen.getByText("Active")).toBeVisible();
});
it("sorts by soonest expiry and displays limit reached and expired states", () => {
  setup([share, { ...share, id: "third", expires_at: "2098-01-01T00:00:00Z", download_count: 3 },
    { ...share, id: "second", expires_at: "2000-01-01T00:00:00Z" }]);
  const rows = screen.getAllByRole("listitem");
  expect(within(rows[0]).getByRole("status")).toHaveTextContent("Expired");
  expect(within(rows[1]).getByRole("status")).toHaveTextContent("Limit reached");
  expect(within(rows[2]).getByRole("status")).toHaveTextContent("Active");
  expect(within(rows[1]).getByText("3 / 3 downloads")).toBeVisible();
  expect(within(rows[2]).getByText(/Copy unavailable/)).toBeVisible();
});
it("copies a known created URL and shows Copied", async () => {
  const client = new QueryClient();
  const url = "https://safedrop.example/s/known";
  vi.mocked(listShares).mockResolvedValue({ items: [share] });
  await rememberShare(client, fileId, { url, expires_at: share.expires_at });
  const writeText = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
  setup([share], client);
  fireEvent.click(screen.getByRole("button", { name: "Copy link" }));
  expect(await screen.findByText("Copied")).toBeVisible();
  expect(writeText).toHaveBeenCalledWith(url);
});
it("does not guess a URL when two links have identical expiry times", async () => {
  const client = new QueryClient();
  vi.mocked(listShares).mockResolvedValue({ items: [share, { ...share, id: "another" }] });
  await rememberShare(client, fileId, { url: "https://safedrop.example/s/new", expires_at: share.expires_at });
  setup([share], client);
  expect(screen.getByRole("button", { name: "Copy link" })).toBeDisabled();
});
