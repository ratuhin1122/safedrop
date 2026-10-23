import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, expect, it, vi } from "vitest";
import { PublicSharePage } from "@/pages/public-share-page";
import { getPublicShare } from "@/api/public-share";
import { downloadShare } from "@/api/downloads";
import { ApiError } from "@/api/client";

vi.mock("@/api/public-share", () => ({ getPublicShare: vi.fn() }));
vi.mock("@/api/downloads", () => ({ downloadShare: vi.fn() }));
const token = "MDEyMzQ1Njc4OWFiY2RlZg";
const info = { name: "holiday.webp", mime: "image/webp", size: 1024, expires_at: new Date(Date.now() + 86400000).toISOString() };
beforeEach(() => vi.clearAllMocks());
function setup() {
  return render(<MemoryRouter initialEntries={["/s/" + token]}><Routes>
    <Route path="/s/:token" element={<PublicSharePage />} />
  </Routes></MemoryRouter>);
}
it("200 shows metadata without consuming a download on load", async () => {
  vi.mocked(getPublicShare).mockResolvedValue(info);
  const view = setup();
  expect(await screen.findByRole("heading", { name: "holiday.webp" })).toBeVisible();
  expect(screen.getByText("1 KB")).toBeVisible();
  expect(screen.getByText(/Expires in/)).toBeVisible();
  expect(screen.getByRole("button", { name: "Download" })).toBeVisible();
  expect(downloadShare).not.toHaveBeenCalled();
  expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
  expect(document.querySelector('meta[name="robots"]')).toHaveAttribute("content", "noindex, nofollow, noarchive");
  view.unmount();
  expect(document.querySelector('meta[name="robots"]')).not.toBeInTheDocument();
});
it("401 prompts for a password and unlocks without login", async () => {
  vi.mocked(getPublicShare).mockRejectedValueOnce(new ApiError("password_required", "Password required", 401)).mockResolvedValueOnce(info);
  setup();
  expect(await screen.findByRole("heading", { name: "This file is password protected" })).toBeVisible();
  fireEvent.change(screen.getByLabelText("Password"), { target: { value: "secret" } });
  fireEvent.click(screen.getByRole("button", { name: "Unlock file" }));
  expect(await screen.findByRole("heading", { name: "holiday.webp" })).toBeVisible();
  expect(getPublicShare).toHaveBeenLastCalledWith(token, "secret", expect.any(AbortSignal));
});
it.each([[404, "Link not found"], [410, "This link has expired or reached its download limit"]] as const)(
  "%s shows its own friendly screen", async (status, title) => {
    vi.mocked(getPublicShare).mockRejectedValue(new ApiError("unavailable", "Unavailable", status));
    setup();
    expect(await screen.findByRole("heading", { name: title })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Download" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Password")).not.toBeInTheDocument();
  });
it("wrong passwords remain on the password screen", async () => {
  vi.mocked(getPublicShare).mockRejectedValue(new ApiError("password_required", "Wrong password", 401));
  setup();
  await screen.findByLabelText("Password");
  fireEvent.change(screen.getByLabelText("Password"), { target: { value: "wrong" } });
  fireEvent.click(screen.getByRole("button", { name: "Unlock file" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("That password didn't work");
});
