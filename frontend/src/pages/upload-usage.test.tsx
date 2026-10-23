import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { expect, it, vi } from "vitest";
import { UploadPage } from "@/pages/upload-page";
import { setAccessToken } from "@/api/session";
import { getUsage } from "@/api/usage";

vi.mock("@/api/usage", () => ({ getUsage: vi.fn() }));
const usage = { bytes_today: 100, bytes_limit: 100, uploads_this_hour: 5, uploads_limit: 20 };
function setup() {
  setAccessToken("test-token");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  render(<QueryClientProvider client={client}><UploadPage /></QueryClientProvider>);
  return client;
}
it("blocks drops at full quota and reopens after reset", async () => {
  vi.mocked(getUsage).mockResolvedValue(usage);
  const client = setup();
  expect(await screen.findByText("Daily limit reached")).toBeVisible();
  expect(screen.getByRole("button", { name: "Browse files" })).toBeDisabled();
  fireEvent.drop(screen.getByRole("region", { name: "File drop zone" }),
    { dataTransfer: { files: [new File(["png"], "blocked.png")] } });
  expect(screen.queryByText("blocked.png")).not.toBeInTheDocument();
  expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "100");
  act(() => { client.setQueryData(["usage"], { ...usage, bytes_today: 0 }); });
  await waitFor(() => expect(screen.getByRole("button", { name: "Browse files" })).toBeEnabled());
  client.clear();
});
it("shows specific fetch errors", async () => {
  vi.mocked(getUsage).mockRejectedValue(new Error("Service unavailable"));
  const client = setup();
  expect(await screen.findByRole("alert")).toHaveTextContent("Could not load usage: Service unavailable");
  expect(screen.getByRole("button", { name: "Refresh usage" })).toBeEnabled();
  client.clear();
});
