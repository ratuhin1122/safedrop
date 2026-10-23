import { useQuery } from "@tanstack/react-query";
import { listShares } from "@/api/shares";
import { Button } from "@/components/ui/button";
import { ShareList } from "@/components/share-list";
import { ToastProvider } from "@/components/toast-provider";

export function FileShares({ fileId }: { fileId: string }) {
  const query = useQuery({ queryKey: ["shares", fileId], queryFn: ({ signal }) => listShares(fileId, signal), refetchInterval: 30000 });
  return <section aria-label="Share links" className="mt-6">
    <h3 className="font-medium">Shared links</h3>
    {query.isPending && <p role="status" className="mt-3 text-sm">Loading share links…</p>}
    {query.error && <div role="alert" className="mt-3 text-sm text-destructive">
      Could not load share links: {query.error.message}
      <Button variant="outline" onClick={() => { void query.refetch(); }}>Retry share links</Button>
    </div>}
    {!query.isPending && <ToastProvider><ShareList shares={query.data?.items ?? []} fileId={fileId} /></ToastProvider>}
  </section>;
}
