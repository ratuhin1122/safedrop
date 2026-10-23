import { useInfiniteQuery } from "@tanstack/react-query";
import { listFiles } from "@/api/files";

export function useFiles() {
  return useInfiniteQuery({
    queryKey: ["files"], initialPageParam: "",
    queryFn: ({ pageParam, signal }) => listFiles({ cursor: pageParam || undefined, limit: 24 }, signal),
    getNextPageParam: (page, _pages, cursor, cursors) =>
      page.next_cursor && page.next_cursor !== cursor && !cursors.includes(page.next_cursor)
        ? page.next_cursor : undefined,
  });
}
