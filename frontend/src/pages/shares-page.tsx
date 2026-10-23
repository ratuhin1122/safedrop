import { useEffect } from "react";
import { useQueries } from "@tanstack/react-query";
import { Link } from "react-router";
import { useFiles } from "@/hooks/use-files";
import { listShares } from "@/api/shares";
import { ShareList } from "@/components/share-list";
import { Button } from "@/components/ui/button";

export function SharesPage() {
  const files = useFiles();
  const items = [...new Map(files.data?.pages.flatMap((page) => page.items).map((file) => [file.id, file])).values()];
  const queries = useQueries({ queries: items.map((file) => ({
    queryKey: ["shares", file.id], queryFn: ({ signal }: { signal: AbortSignal }) => listShares(file.id, signal),
    refetchInterval: 30000,
  })) });
  const { hasNextPage, isFetchingNextPage, isFetchNextPageError, fetchNextPage } = files;
  // The API lists links per file, so discover every owned file before declaring the list complete.
  useEffect(() => {
    if (hasNextPage && !isFetchingNextPage && !isFetchNextPageError) void fetchNextPage();
  }, [hasNextPage, isFetchingNextPage, isFetchNextPageError, fetchNextPage]);
  const loading = files.isPending || hasNextPage || queries.some((query) => query.isPending);
  const shares = queries.flatMap((query) => query.data?.items ?? []);
  const errors = [files.error, ...queries.map((query) => query.error)].filter((error) => error !== null);
  return <section className="mx-auto max-w-3xl" aria-labelledby="shares-title">
    <h1 id="shares-title" className="text-3xl font-semibold tracking-tight">Shared links</h1>
    <p className="mb-6 mt-3 text-sm text-muted-foreground">Manage access to your files. Soonest expiry comes first.</p>
    {loading && <p role="status" className="mb-4 text-sm text-muted-foreground">Loading your shared links…</p>}
    {errors.length > 0 && <div role="alert" className="mb-4 space-y-3 rounded-lg border p-4 text-sm text-destructive">
      <p>Some links could not be loaded: {errors[0]?.message}</p>
      <Button variant="outline" onClick={() => {
        if (files.isFetchNextPageError) void files.fetchNextPage();
        else if (files.error) void files.refetch();
        queries.forEach((query) => { if (query.error) void query.refetch(); });
      }}>Try again</Button>
    </div>}
    {(!loading || shares.length > 0) && <ShareList shares={shares} names={Object.fromEntries(items.map((file) => [file.id, file.original_name]))} />}
    {!loading && !errors.length && !shares.length && <Link to="/files" className="mt-5 inline-block rounded-sm text-sm text-primary underline focus-visible:outline-2">Choose a file to share</Link>}
  </section>;
}
