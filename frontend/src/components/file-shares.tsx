import { useQuery } from "@tanstack/react-query";
import { listShares } from "@/api/shares";
import { Button } from "@/components/ui/button";

export function FileShares({ fileId }: { fileId: string }) {
  const query = useQuery({ queryKey: ["shares", fileId], queryFn: ({ signal }) => listShares(fileId, signal) });
  return <section aria-label="Share links" className="mt-6 border-t pt-5">
    <h3 className="font-medium">Share links</h3>
    {query.isPending && <p role="status" className="mt-3 text-sm">Loading share links…</p>}
    {query.error && <div role="alert" className="mt-3 text-sm text-destructive">
      Could not load share links: {query.error.message}
      <Button variant="outline" onClick={() => { void query.refetch(); }}>Retry share links</Button>
    </div>}
    {query.data?.items.length === 0 && <p className="mt-3 text-sm text-muted-foreground">No share links yet.</p>}
    <ul className="mt-3 space-y-3">{query.data?.items.map((share) => <li key={share.id} className="rounded-md border p-3 text-sm">
      <p>{Date.parse(share.expires_at) <= Date.now() ? "Expired" : "Expires"} {new Date(share.expires_at).toLocaleString()}</p>
      <p className="mt-1 text-xs text-muted-foreground">{share.download_count} downloads{share.max_downloads !== null && " / " + share.max_downloads}
        {share.password_protected && " · Password protected"}</p>
    </li>)}</ul>
  </section>;
}
