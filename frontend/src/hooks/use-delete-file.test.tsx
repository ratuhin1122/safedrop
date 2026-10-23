import { act, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider, useQuery, type InfiniteData } from "@tanstack/react-query";
import { expect, it, vi } from "vitest";
import { useDeleteFile } from "@/hooks/use-delete-file";
import { ToastContext, type ToastInput } from "@/hooks/use-toast";
import { deleteFile } from "@/api/files";
import { fileFixture } from "@/api/fixtures.test-data";
import type { FileItem, FilePage } from "@/api/schemas";

vi.mock("@/api/files", () => ({ deleteFile: vi.fn() }));
const file: FileItem = { ...fileFixture, status: "ready" };
function setup() {
  const notices: ToastInput[] = [];
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(["files"], { pages: [{ items: [file], next_cursor: "" }], pageParams: [""] });
  function Harness() {
    const remove = useDeleteFile();
    const query = useQuery<InfiniteData<FilePage>>({ queryKey: ["files"], enabled: false });
    return <><button onClick={() => { void remove(file); }}>Remove</button>
      <span>{query.data?.pages.flatMap((page) => page.items).length} files</span></>;
  }
  render(<QueryClientProvider client={client}><ToastContext.Provider value={(notice) => notices.push(notice)}>
    <Harness />
  </ToastContext.Provider></QueryClientProvider>);
  return { client, notices };
}
it("removes immediately, then restores on server failure", async () => {
  vi.mocked(deleteFile).mockRejectedValue(new Error("Storage unavailable"));
  const { client, notices } = setup();
  await act(async () => { screen.getByText("Remove").click(); });
  expect(client.getQueryData<InfiniteData<FilePage>>(["files"])?.pages[0].items).toHaveLength(0);
  expect(deleteFile).not.toHaveBeenCalled();
  await act(async () => notices[0].onDismiss?.());
  expect(client.getQueryData<InfiniteData<FilePage>>(["files"])?.pages[0].items).toEqual([file]);
  expect(notices[1].description).toBe("Storage unavailable");
  client.clear();
});
it("Undo restores the file without sending DELETE", async () => {
  vi.mocked(deleteFile).mockClear();
  const { client, notices } = setup();
  await act(async () => { screen.getByText("Remove").click(); });
  await act(async () => { notices[0].action?.onClick(); notices[0].onDismiss?.(); });
  expect(client.getQueryData<InfiniteData<FilePage>>(["files"])?.pages[0].items).toEqual([file]);
  expect(deleteFile).not.toHaveBeenCalled();
  client.clear();
});
