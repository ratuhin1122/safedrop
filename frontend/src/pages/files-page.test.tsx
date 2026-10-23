import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "vitest";
import { FilesPage } from "@/pages/files-page";
import { listFiles } from "@/api/files";
import { fileFixture } from "@/api/fixtures.test-data";

vi.mock("@/api/files", () => ({ listFiles: vi.fn() }));
vi.mock("@/components/file-thumbnail", () => ({ FileThumbnail: () => <span>Preview</span> }));
function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  return render(<QueryClientProvider client={client}><MemoryRouter><FilesPage /></MemoryRouter></QueryClientProvider>);
}
it("scrolls with next_cursor, filters loaded names, and toggles views", async () => {
  let intersect!: IntersectionObserverCallback;
  const disconnect = vi.fn();
  vi.stubGlobal("IntersectionObserver", class {
    constructor(callback: IntersectionObserverCallback) { intersect = callback; }
    observe = vi.fn(); disconnect = disconnect;
  });
  vi.mocked(listFiles).mockResolvedValueOnce({ items: [{ ...fileFixture, status: "ready" }], next_cursor: "cursor-two" })
    .mockResolvedValueOnce({ items: [{ ...fileFixture, id: "22222222-2222-4222-8222-222222222222", original_name: "sunset.png", status: "processing" }], next_cursor: "" });
  setup();
  expect(await screen.findByText("image.jpg")).toBeVisible();
  await act(async () => { intersect([{ isIntersecting: true } as IntersectionObserverEntry], {} as IntersectionObserver); });
  expect(await screen.findByText("sunset.png")).toBeVisible();
  expect(listFiles).toHaveBeenNthCalledWith(2, { cursor: "cursor-two", limit: 24 }, expect.any(AbortSignal));
  expect(screen.queryByRole("button", { name: "Load more files" })).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "List" }));
  expect(screen.getByRole("button", { name: "List" })).toHaveAttribute("aria-pressed", "true");
  fireEvent.change(screen.getByRole("searchbox"), { target: { value: "SUNSET" } });
  expect(screen.queryByText("image.jpg")).not.toBeInTheDocument();
  expect(screen.getByText("sunset.png")).toBeVisible();
  expect(disconnect).toHaveBeenCalled();
});
it("shows skeletons then a friendly empty state", async () => {
  let finish!: (page: { items: []; next_cursor: string }) => void;
  vi.mocked(listFiles).mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
  setup();
  expect(screen.getByRole("status", { name: "Loading files" })).toBeVisible();
  await act(async () => finish({ items: [], next_cursor: "" }));
  expect(await screen.findByRole("link", { name: "Upload a file" })).toHaveAttribute("href", "/");
});
it("shows a specific error and retries", async () => {
  vi.mocked(listFiles).mockRejectedValueOnce(new Error("Database unavailable"))
    .mockResolvedValueOnce({ items: [], next_cursor: "" });
  setup();
  expect(await screen.findByRole("alert")).toHaveTextContent("Database unavailable");
  fireEvent.click(screen.getByRole("button", { name: "Try again" }));
  await waitFor(() => expect(screen.getByText("Your space is ready.")).toBeVisible());
});
