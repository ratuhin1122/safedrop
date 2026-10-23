import { fireEvent, render, screen, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, expect, it, vi } from "vitest";
import { FilePanel } from "@/components/file-panel";
import { ToastContext } from "@/hooks/use-toast";
import { fileFixture } from "@/api/fixtures.test-data";
import { variantBlob } from "@/api/variant-media";

vi.mock("@/api/variant-media", () => ({ variantBlob: vi.fn(async () => new Blob(["webp"])) }));
vi.mock("@/api/shares", () => ({ listShares: vi.fn(async () => ({ items: [] })) }));
beforeEach(() => {
  URL.createObjectURL ??= () => "blob:preview";
  URL.revokeObjectURL ??= () => {};
  HTMLDialogElement.prototype.showModal = function () { this.setAttribute("open", ""); };
  HTMLDialogElement.prototype.close = function () { this.removeAttribute("open"); };
});
function setup(rejected = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  return render(<QueryClientProvider client={client}><ToastContext.Provider value={vi.fn()}>
    <FilePanel file={{ ...fileFixture, variants: { ...fileFixture.variants, w768: { ...fileFixture.variants.w375, width: 768 }, w1200: { ...fileFixture.variants.w375, width: 1200 } }, status: rejected ? "rejected" : "ready", reject_reason: "Virus found" }}
      onClose={vi.fn()} />
  </ToastContext.Provider></QueryClientProvider>);
}
it("shows metadata, three authenticated preview sources, shares, and a confirmation", async () => {
  const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:preview");
  const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  const view = setup();
  expect(screen.getByText("image/webp")).toBeVisible();
  expect(await screen.findByAltText("image.jpg")).toHaveAttribute("srcset", "blob:preview 375w, blob:preview 768w, blob:preview 1200w");
  fireEvent.click(screen.getByRole("tab", { name: "Shared links" }));
  expect(await screen.findByText("No share links yet.")).toBeVisible();
  fireEvent.click(screen.getByRole("tab", { name: "Details" }));
  expect(screen.getByRole("button", { name: "Download" })).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "Delete file" }));
  expect(within(screen.getByRole("dialog", { name: "Confirm deletion" })).getByRole("button", { name: "Keep file" })).toBeVisible();
  view.unmount();
  expect(revoke).toHaveBeenCalled();
  create.mockRestore(); revoke.mockRestore();
});
it("rejected files show the reason and deletion only", () => {
  vi.mocked(variantBlob).mockClear();
  setup(true);
  expect(screen.getByText("Virus found")).toBeVisible();
  expect(screen.queryByRole("button", { name: "Download" })).not.toBeInTheDocument();
  expect(screen.queryByRole("region", { name: "Share links" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Delete file" })).toBeVisible();
  expect(variantBlob).not.toHaveBeenCalled();
});
