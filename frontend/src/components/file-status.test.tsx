import { act, cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { FileStatus } from "@/components/file-status";
import { getFile } from "@/api/files";
import { fileFixture, fileId } from "@/api/fixtures.test-data";
import { fileSchema, type FileItem } from "@/api/schemas";

vi.mock("@/api/files", () => ({ getFile: vi.fn() }));
let visibility: DocumentVisibilityState;
let client: QueryClient;
const file = (status: FileItem["status"], reject_reason: string | null = null) => fileSchema.parse({ ...fileFixture, status, reject_reason });

beforeEach(() => {
  vi.useFakeTimers();
  visibility = "visible";
  vi.spyOn(document, "visibilityState", "get").mockImplementation(() => visibility);
  notifyManager.setScheduler((callback) => callback());
  vi.mocked(getFile).mockReset();
  client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
});
afterEach(() => {
  cleanup(); client.clear(); vi.useRealTimers(); vi.restoreAllMocks();
  notifyManager.setScheduler((callback) => setTimeout(callback, 0));
});

async function show() {
  await act(async () => {
    render(<QueryClientProvider client={client}><FileStatus id={fileId} name="image.jpg" /></QueryClientProvider>);
  });
}
async function tick(milliseconds: number) {
  await act(async () => { await vi.advanceTimersByTimeAsync(milliseconds); });
}
function currentStep() {
  return within(screen.getByRole("list", { name: "File processing steps" })).getAllByRole("listitem")
    .find((item) => item.getAttribute("aria-current") === "step");
}
async function setVisible(value: DocumentVisibilityState) {
  await act(async () => { visibility = value; document.dispatchEvent(new Event("visibilitychange")); });
}

it("advances the stepper through mocked states and stops polling at ready", async () => {
  vi.mocked(getFile).mockResolvedValueOnce(file("uploaded")).mockResolvedValueOnce(file("scanning"))
    .mockResolvedValueOnce(file("processing")).mockResolvedValue(file("ready"));
  await show();
  expect(currentStep()).toHaveTextContent("Uploaded");
  expect(getFile).toHaveBeenCalledTimes(1);
  await tick(1999);
  expect(getFile).toHaveBeenCalledTimes(1);
  await tick(1);
  expect(currentStep()).toHaveTextContent("Scanning");
  await tick(4000);
  expect(currentStep()).toHaveTextContent("Processing");
  await tick(8000);
  expect(currentStep()).toHaveTextContent("Ready");
  expect(screen.getByRole("status")).toHaveTextContent("Ready to share");
  await tick(60000);
  expect(getFile).toHaveBeenCalledTimes(4);
});

it("backs off to a maximum polling interval of ten seconds", async () => {
  vi.mocked(getFile).mockResolvedValue(file("scanning"));
  await show();
  await tick(2000); await tick(4000); await tick(8000);
  expect(getFile).toHaveBeenCalledTimes(4);
  await tick(9999);
  expect(getFile).toHaveBeenCalledTimes(4);
  await tick(1);
  expect(getFile).toHaveBeenCalledTimes(5);
  await tick(10000);
  expect(getFile).toHaveBeenCalledTimes(6);
});

it("does not fetch while initially hidden and resumes when visible", async () => {
  visibility = "hidden";
  vi.mocked(getFile).mockResolvedValue(file("scanning"));
  await show();
  await tick(30000);
  expect(getFile).not.toHaveBeenCalled();
  await setVisible("visible");
  expect(getFile).toHaveBeenCalledOnce();
  await setVisible("hidden");
  await tick(30000);
  expect(getFile).toHaveBeenCalledOnce();
  await setVisible("visible");
  expect(getFile).toHaveBeenCalledTimes(2);
});

it("aborts a status request when the tab is hidden", async () => {
  let signal: AbortSignal | undefined;
  vi.mocked(getFile).mockImplementation((_id, current) => { signal = current; return new Promise(() => {}); });
  await show();
  expect(signal?.aborted).toBe(false);
  await setVisible("hidden");
  expect(signal?.aborted).toBe(true);
  await tick(30000);
  expect(getFile).toHaveBeenCalledOnce();
});

it.each([
  ["virus", "A virus was found."],
  ["invalid_image", "This image could not be read."],
  ["pixel_limit", "This image's dimensions are too large."],
])("shows a friendly %s rejection and stops polling", async (reason, message) => {
  vi.mocked(getFile).mockResolvedValue(file("rejected", reason));
  await show();
  expect(screen.getByRole("alert")).toHaveTextContent(message);
  expect(screen.queryByRole("list", { name: "File processing steps" })).not.toBeInTheDocument();
  await tick(30000);
  expect(getFile).toHaveBeenCalledOnce();
});
