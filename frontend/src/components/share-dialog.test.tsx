import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, expect, it, vi } from "vitest";
import QRCode from "qrcode";
import { ShareDialog } from "@/components/share-dialog";
import { createShare } from "@/api/shares";
import { fileId } from "@/api/fixtures.test-data";

vi.mock("@/api/shares", () => ({ createShare: vi.fn() }));
vi.mock("qrcode", () => ({ default: { toDataURL: vi.fn(async () => "data:image/png;base64,cXI=") } }));
beforeEach(() => {
  vi.clearAllMocks();
  HTMLDialogElement.prototype.showModal = function () { this.setAttribute("open", ""); };
  HTMLDialogElement.prototype.close = function () { this.removeAttribute("open"); };
});
function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  render(<QueryClientProvider client={client}><ShareDialog fileId={fileId} onClose={vi.fn()} /></QueryClientProvider>);
}
function response() {
  return { url: "https://safedrop.example/s/abcdefghijklmnopqrstuv", expires_at: new Date(Date.now() + 7 * 86400000).toISOString() };
}
it("creates a seven-day link, shows a local QR code, and copies it with a Copied toast", async () => {
  const result = response();
  vi.mocked(createShare).mockResolvedValue(result);
  const writeText = vi.fn().mockResolvedValue(undefined);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
  setup();
  fireEvent.click(screen.getByRole("button", { name: "Create link" }));
  expect(await screen.findByDisplayValue(result.url)).toBeVisible();
  expect(screen.getByText("Expires in 7 days")).toBeVisible();
  const input = vi.mocked(createShare).mock.calls[0][1];
  expect(input).not.toHaveProperty("password");
  expect(input).not.toHaveProperty("max_downloads");
  expect(Date.parse(input.expires_at) - Date.now()).toBeGreaterThan(6.99 * 86400000);
  expect(await screen.findByAltText("QR code for your share link")).toBeVisible();
  expect(QRCode.toDataURL).toHaveBeenCalledWith(result.url, expect.objectContaining({ margin: 4 }));
  fireEvent.click(screen.getByRole("button", { name: "Copy link" }));
  expect(await screen.findByText("Copied")).toBeVisible();
  expect(writeText).toHaveBeenCalledWith(result.url);
});
it("rejects past custom dates and nonpositive download limits", async () => {
  setup();
  fireEvent.click(screen.getByLabelText("Custom date"));
  fireEvent.change(screen.getByLabelText("Expiry date and time"), { target: { value: "2000-01-01T12:00" } });
  fireEvent.change(screen.getByLabelText("Max downloads (optional)"), { target: { value: "0" } });
  fireEvent.click(screen.getByRole("button", { name: "Create link" }));
  expect(await screen.findByText("Choose a date and time in the future.")).toBeVisible();
  expect(screen.getByText("Enter a positive whole number of downloads.")).toBeVisible();
  expect(createShare).not.toHaveBeenCalled();
});
it("submits custom expiry and optional restrictions and reports clipboard errors", async () => {
  vi.mocked(createShare).mockResolvedValue(response());
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: vi.fn().mockRejectedValue(new Error("Denied")) } });
  setup();
  fireEvent.click(screen.getByLabelText("Custom date"));
  fireEvent.change(screen.getByLabelText("Expiry date and time"), { target: { value: "2099-01-01T12:00" } });
  fireEvent.change(screen.getByLabelText("Password (optional)"), { target: { value: "secret passphrase" } });
  fireEvent.change(screen.getByLabelText("Max downloads (optional)"), { target: { value: "12" } });
  fireEvent.click(screen.getByRole("button", { name: "Create link" }));
  await waitFor(() => expect(createShare).toHaveBeenCalledWith(fileId, {
    expires_at: new Date("2099-01-01T12:00").toISOString(), password: "secret passphrase", max_downloads: 12,
  }, expect.any(AbortSignal)));
  fireEvent.click(await screen.findByRole("button", { name: "Copy link" }));
  expect(await screen.findByText(/Could not copy the link/)).toBeVisible();
  expect(screen.queryByText("Copied")).not.toBeInTheDocument();
});
it("shows specific API errors and lets the user try again", async () => {
  vi.mocked(createShare).mockRejectedValue(new Error("File is no longer ready"));
  setup();
  fireEvent.click(screen.getByRole("button", { name: "Create link" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("File is no longer ready");
  expect(screen.getByRole("button", { name: "Create link" })).toBeEnabled();
});
