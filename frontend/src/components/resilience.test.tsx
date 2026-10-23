import { act, fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { ErrorBoundary } from "@/components/error-boundary";
import { OfflineBanner } from "@/components/offline-banner";
import { UploadDropZone } from "@/components/upload-drop-zone";
import { UploadRow } from "@/components/upload-row";

it("shows a recovery screen when a component crashes", () => {
  const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
  function Broken(): never { throw new Error("private implementation details"); }
  render(<ErrorBoundary><Broken /></ErrorBoundary>);
  expect(screen.getByRole("heading", { name: "Something went wrong" })).toBeVisible();
  expect(screen.getByRole("button", { name: "Reload page" })).toBeVisible();
  expect(screen.queryByText("private implementation details")).not.toBeInTheDocument();
  consoleError.mockRestore();
});
it("announces offline connectivity and removes the banner on reconnect", () => {
  const online = vi.spyOn(navigator, "onLine", "get").mockReturnValue(true);
  render(<OfflineBanner />);
  expect(screen.queryByText(/You’re offline/)).not.toBeInTheDocument();
  act(() => { online.mockReturnValue(false); window.dispatchEvent(new Event("offline")); });
  expect(screen.getByText(/You’re offline/).parentElement).toHaveAttribute("aria-live", "polite");
  act(() => { online.mockReturnValue(true); window.dispatchEvent(new Event("online")); });
  expect(screen.queryByText(/You’re offline/)).not.toBeInTheDocument();
  online.mockRestore();
});
it("opens the picker with Alt+Shift+U, ignores editors, and respects disabled state", () => {
  const { rerender } = render(<UploadDropZone maxBytes={100} onFiles={vi.fn()} />);
  const input = screen.getByLabelText("Choose files");
  const click = vi.spyOn(input, "click").mockImplementation(() => {});
  fireEvent.keyDown(window, { key: "U", altKey: true, shiftKey: true });
  expect(click).toHaveBeenCalledOnce();
  const editor = document.createElement("textarea"); document.body.appendChild(editor);
  fireEvent.keyDown(editor, { key: "u", altKey: true, shiftKey: true });
  expect(click).toHaveBeenCalledOnce(); editor.remove();
  rerender(<UploadDropZone maxBytes={100} onFiles={vi.fn()} disabled />);
  fireEvent.keyDown(window, { key: "u", altKey: true, shiftKey: true });
  expect(click).toHaveBeenCalledOnce();
  click.mockRestore();
});
it("announces progress in ten-percent steps and errors assertively", () => {
  const item = { id: 1, file: new File(["png"], "photo.png"), error: null, state: "uploading" as const, progress: 47 };
  const props = { cooldown: 0, onRemove: vi.fn(), onCancel: vi.fn(), onRetry: vi.fn() };
  const { container, rerender } = render(<UploadRow item={item} {...props} />);
  expect(container.querySelector('[aria-live="polite"]')).toHaveTextContent("photo.png: Uploading 40%");
  rerender(<UploadRow item={{ ...item, state: "error", error: "File too large" }} {...props} />);
  expect(screen.getByRole("alert")).toHaveAttribute("aria-live", "assertive");
  expect(screen.getByRole("alert")).toHaveTextContent("photo.png: File too large");
});
