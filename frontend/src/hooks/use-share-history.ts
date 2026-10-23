import { getAccessToken } from "@/api/session";
import { useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { listShares } from "@/api/shares";
import type { ShareItem } from "@/api/schemas";

type History = { urls: Record<string, string>; revoked: Record<string, ShareItem> };
const key = ["share-history"];
export function useShareHistory() {
  const client = useQueryClient();
  const query = useQuery<History>({ queryKey: key, queryFn: () => ({ urls: {}, revoked: {} }),
    initialData: { urls: {}, revoked: {} }, enabled: false, gcTime: Infinity });
  const markRevoked = (share: ShareItem) => client.setQueryData<History>(key, (current) => ({
    urls: current?.urls ?? {}, revoked: { ...current?.revoked, [share.id]: share },
  }));
  return { ...query.data!, markRevoked };
}
export async function rememberShare(client: QueryClient, fileId: string, created: { url: string; expires_at: string }) {
  const session = getAccessToken();
  const response = await listShares(fileId);
  if (session !== getAccessToken()) return;
  const matching = response.items.filter((share) => Date.parse(share.expires_at) === Date.parse(created.expires_at));
  // The contract has no ID in the create response. Never guess between matching links.
  if (matching.length !== 1) return;
  client.setQueryData<History>(key, (current) => ({
    revoked: current?.revoked ?? {}, urls: { ...current?.urls, [matching[0].id]: created.url },
  }));
}
