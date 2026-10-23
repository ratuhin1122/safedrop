import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { UploadPage } from "@/pages/upload-page";

function file(name: string, size: number, type = "image/png") {
  const value = new File(["fixture"], name, { type });
  // Size-only fixture: do not allocate 100 MB to verify a metadata pre-check.
  Object.defineProperty(value, "size", { value: size });
  return value;
}

function drop(files: File[]) {
  fireEvent.drop(screen.getByRole("region", { name: "File drop zone" }), { dataTransfer: { files, types: ["Files"] } });
}

it("shows types and the maximum before selecting anything", () => {
  render(<UploadPage />);
  expect(screen.getByText(/JPEG, PNG, WebP and HEIC/)).toBeVisible();
  expect(screen.getByText(/Max size is 25 MB per file/)).toBeVisible();
  expect(screen.getByRole("button", { name: "Browse files" })).toBeVisible();
  expect(screen.queryByRole("list", { name: "File queue" })).not.toBeInTheDocument();
});

it('rejects a dropped 100 MB file with "Max size is 25 MB" and never uploads', () => {
  const fetcher = vi.fn();
  vi.stubGlobal("fetch", fetcher);
  render(<UploadPage />);
  drop([file("large.png", 100 * 1024 ** 2)]);
  expect(screen.getByText("Max size is 25 MB")).toBeVisible();
  expect(screen.getByText("large.png")).toBeVisible();
  expect(screen.getByRole("status")).toHaveTextContent("0 selected · 1 not added");
  expect(fetcher).not.toHaveBeenCalled();
});

it("handles mixed selections independently and treats extensions only as hints", async () => {
  const fetcher = vi.fn();
  vi.stubGlobal("fetch", fetcher);
  render(<UploadPage />);
  drop([file("photo.JPG", 100, "text/plain"), file("report.pdf", 20), file("empty.png", 0)]);
  expect(screen.getByRole("status")).toHaveTextContent("1 selected · 2 not added");
  expect(screen.getByText(/Unsupported file extension/)).toBeVisible();
  expect(screen.getByText("This file is empty. Choose a file with content.")).toBeVisible();
  const queue = screen.getByRole("list", { name: "File queue" });
  expect(within(queue).getAllByRole("listitem")).toHaveLength(3);
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Remove photo.JPG" }));
  expect(screen.queryByText("photo.JPG")).not.toBeInTheDocument();
  expect(fetcher).not.toHaveBeenCalled();
});

it("supports browsing, repeated selections, and removal without requests", async () => {
  const user = userEvent.setup({ applyAccept: false });
  const fetcher = vi.fn();
  vi.stubGlobal("fetch", fetcher);
  render(<UploadPage />);
  const input = screen.getByLabelText("Choose files");
  const openPicker = vi.spyOn(input, "click");
  await user.click(screen.getByRole("button", { name: "Browse files" }));
  expect(openPicker).toHaveBeenCalledOnce();
  const selected = file("tiny.webp", 1024);
  await user.upload(input, [selected, file("other.heic", 2048)]);
  await user.upload(input, selected);
  expect(screen.getByRole("status")).toHaveTextContent("3 selected");
  expect(screen.getAllByText("tiny.webp")).toHaveLength(2);
  openPicker.mockRestore();
  expect(fetcher).not.toHaveBeenCalled();
});

it("highlights the drop zone until the drag leaves or drops", () => {
  render(<UploadPage />);
  const zone = screen.getByRole("region", { name: "File drop zone" });
  fireEvent.dragEnter(zone, { dataTransfer: { types: ["Files"] } });
  expect(zone).toHaveClass("bg-accent");
  expect(screen.getByText("Drop your files here")).toBeVisible();
  fireEvent.dragLeave(zone);
  expect(zone).not.toHaveClass("bg-accent");
  fireEvent.dragEnter(zone, { dataTransfer: { types: ["Files"] } });
  drop([file("photo.png", 10)]);
  expect(zone).not.toHaveClass("bg-accent");
});

it("uses the configured limit, accepts its exact boundary, and fails closed for invalid config", () => {
  vi.stubEnv("VITE_MAX_UPLOAD_BYTES", "1048576");
  const view = render(<UploadPage />);
  expect(screen.getByText(/Max size is 1 MB per file/)).toBeVisible();
  drop([file("boundary.png", 1048576), file("too-large.png", 1048577)]);
  expect(screen.getByRole("status")).toHaveTextContent("1 selected · 1 not added");
  view.unmount();
  vi.stubEnv("VITE_MAX_UPLOAD_BYTES", "not-a-size");
  render(<UploadPage />);
  expect(screen.getByRole("alert")).toHaveTextContent("VITE_MAX_UPLOAD_BYTES must be a positive whole number of bytes.");
  expect(screen.queryByRole("button", { name: "Browse files" })).not.toBeInTheDocument();
});
