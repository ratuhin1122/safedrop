import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { UploadPage } from "@/pages/upload-page";
import { MockXHR } from "@/api/mock-xhr.test-data";
import { setAccessToken } from "@/api/session";

const response = { id: "11111111-1111-4111-8111-111111111111", status: "uploaded" };

beforeEach(() => {
  MockXHR.requests = [];
  vi.stubGlobal("XMLHttpRequest", MockXHR);
  vi.stubEnv("VITE_API_URL", "/v1");
  setAccessToken("test-jwt");
});
afterEach(() => vi.useRealTimers());

function select(count: number) {
  render(<UploadPage />);
  const files = Array.from({ length: count }, (_, i) => new File(["png"], `photo-${i}.png`, { type: "image/png" }));
  fireEvent.drop(screen.getByRole("region", { name: "File drop zone" }), { dataTransfer: { files } });
  fireEvent.click(screen.getByRole("button", { name: "Upload files" }));
}

it("uses XHR with a JWT and never exceeds three simultaneous uploads", async () => {
  select(5);
  expect(MockXHR.requests).toHaveLength(3);
  expect(MockXHR.requests[0].headers.get("Authorization")).toBe("Bearer test-jwt");
  expect(MockXHR.requests[0].headers.has("Content-Type")).toBe(false);
  expect(MockXHR.requests[0].body?.getAll("file")).toHaveLength(1);
  act(() => MockXHR.requests[0].progress(40));
  expect(screen.getByRole("progressbar", { name: "Upload progress for photo-0.png" })).toHaveAttribute("value", "40");
  await act(async () => MockXHR.requests[0].respond(201, response));
  expect(MockXHR.requests).toHaveLength(4);
  expect(screen.getAllByText(/Uploaded, checking file/).length).toBeGreaterThan(0);
  await act(async () => MockXHR.requests[1].respond(201, response));
  expect(MockXHR.requests).toHaveLength(5);
});

it("counts down a mocked 429 and disables new uploads and retry until expiry", async () => {
  vi.useFakeTimers();
  select(4);
  await act(async () => MockXHR.requests[0].respond(429, { code: "rate_limited", message: "Slow down." }, "3"));
  fireEvent.drop(screen.getByRole("region", { name: "File drop zone" }), {
    dataTransfer: { files: [new File(["png"], "new.png")] },
  });
  expect(screen.getAllByText("Too many uploads, try again in 3 seconds").length).toBeGreaterThan(0);
  expect(screen.getByRole("button", { name: "Retry photo-0.png" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Upload files" })).toBeDisabled();
  await act(async () => { vi.advanceTimersByTime(1000); });
  expect(screen.getAllByText("Too many uploads, try again in 2 seconds").length).toBeGreaterThan(0);
  expect(MockXHR.requests).toHaveLength(3);
  await act(async () => { vi.advanceTimersByTime(2000); });
  expect(MockXHR.requests).toHaveLength(4);
  expect(screen.getByRole("button", { name: "Retry photo-0.png" })).toBeEnabled();
  expect(screen.getByRole("button", { name: "Upload files" })).toBeEnabled();
  fireEvent.click(screen.getByRole("button", { name: "Retry photo-0.png" }));
  expect(MockXHR.requests).toHaveLength(4); // Three slots are still occupied.
  await act(async () => MockXHR.requests[1].respond(201, response));
  expect(MockXHR.requests).toHaveLength(5);
});

it("cancels an active request, frees its slot, and allows a manual retry", async () => {
  select(4);
  await act(async () => fireEvent.click(screen.getByRole("button", { name: "Cancel photo-0.png" })));
  expect(MockXHR.requests[0].aborted).toBe(true);
  expect(MockXHR.requests).toHaveLength(4);
  fireEvent.click(screen.getByRole("button", { name: "Retry photo-0.png" }));
  expect(MockXHR.requests).toHaveLength(4);
  await act(async () => MockXHR.requests[1].respond(201, response));
  expect(MockXHR.requests).toHaveLength(5);
});

it.each([
  [413, "This file is too large. Choose a smaller file."],
  [415, "Unsupported file type. Choose JPEG, PNG, WebP or HEIC."],
  [422, "Image dimensions are too large. Choose a smaller image."],
])("shows a friendly error for HTTP %s", async (status, message) => {
  select(1);
  await act(async () => MockXHR.requests[0].respond(status, { code: "rejected", message: "Server detail" }));
  expect(screen.getByText(message)).toBeVisible();
  expect(screen.getByRole("button", { name: "Retry photo-0.png" })).toBeEnabled();
});

it("aborts all active uploads on unmount and does not start waiting files", async () => {
  select(5);
  const queue = screen.getByRole("list", { name: "File queue" });
  const waiting = within(queue).getAllByRole("listitem")[4];
  fireEvent.click(within(waiting).getByRole("button", { name: "Cancel photo-4.png" }));
  const { cleanup } = await import("@testing-library/react");
  await act(async () => cleanup());
  expect(MockXHR.requests).toHaveLength(3);
  expect(MockXHR.requests.every((xhr) => xhr.aborted)).toBe(true);
});
