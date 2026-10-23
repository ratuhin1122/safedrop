import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { getFile } from "@/api/files";
import { ApiError } from "@/api/client";
import { usePageVisible } from "@/hooks/use-page-visible";

const terminal = (status?: string) => status === "ready" || status === "rejected";
const permanentError = (error: unknown) => error instanceof ApiError && [401, 403, 404, 410].includes(error.status);

export function useFileStatus(id: string) {
  const visible = usePageVisible();
  const client = useQueryClient();
  const queryKey = ["file-status", id] as const;
  const query = useQuery({
    queryKey,
    queryFn: ({ signal }) => getFile(id, signal),
    enabled: (current) => Boolean(id) && visible && !terminal(current.state.data?.status) && !permanentError(current.state.error),
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchIntervalInBackground: false,
    refetchInterval: (current) => {
      if (!visible || terminal(current.state.data?.status) || permanentError(current.state.error)) return false;
      const attempts = current.state.dataUpdateCount + current.state.errorUpdateCount;
      const backoff = Math.min(2000 * 2 ** Math.min(3, Math.max(0, attempts - 1)), 10000);
      const error = current.state.error;
      return error instanceof ApiError && error.status === 429
        ? Math.max(backoff, (error.retryAfterSeconds ?? 10) * 1000) : backoff;
    },
  });

  useEffect(() => {
    if (!visible) void client.cancelQueries({ queryKey: ["file-status", id], exact: true });
  }, [visible, client, id]);
  return { ...query, paused: !visible, complete: terminal(query.data?.status) };
}
