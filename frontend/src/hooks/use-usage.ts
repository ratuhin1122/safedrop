import { useCallback } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { getUsage } from "@/api/usage";

export function useUsage() {
  const client = useQueryClient();
  const query = useQuery({
    queryKey: ["usage"], queryFn: ({ signal }) => getUsage(signal),
    staleTime: 15000, refetchInterval: 30000, refetchIntervalInBackground: false, retry: false,
  });
  const refresh = useCallback(() => { void client.invalidateQueries({ queryKey: ["usage"], exact: true }); }, [client]);
  return { ...query, refresh };
}
