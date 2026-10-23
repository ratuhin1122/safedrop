import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { useFiles } from "@/hooks/use-files";
import { FileEntry } from "@/components/file-entry";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function FilesPage() {
  const query = useFiles();
  const [view, setView] = useState<"grid" | "list">("grid");
  const [search, setSearch] = useState("");
  const sentinel = useRef<HTMLDivElement>(null);
  const { fetchNextPage, hasNextPage, isFetchingNextPage, isFetchNextPageError } = query;
  useEffect(() => {
    if (!sentinel.current || !hasNextPage || isFetchingNextPage || isFetchNextPageError
      || typeof IntersectionObserver === "undefined") return;
    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) void fetchNextPage();
    }, { rootMargin: "200px" });
    observer.observe(sentinel.current);
    return () => observer.disconnect();
  }, [fetchNextPage, hasNextPage, isFetchingNextPage, isFetchNextPageError]);
  const items = [...new Map(query.data?.pages.flatMap((page) => page.items).map((file) => [file.id, file])).values()];
  const visible = items.filter((file) => file.original_name.toLocaleLowerCase().includes(search.toLocaleLowerCase().trim()));
  return <section className="mx-auto max-w-5xl" aria-labelledby="files-title">
    <header className="mb-8 flex flex-wrap items-center justify-between gap-4">
      <div><h1 id="files-title" className="text-3xl font-semibold tracking-tight">My files</h1>
        <p className="mt-2 text-sm text-muted-foreground">A quiet home for the things you share.</p></div>
      <div role="group" aria-label="File view" className="flex gap-2">
        <Button variant="outline" aria-pressed={view === "grid"} onClick={() => setView("grid")}>Grid</Button>
        <Button variant="outline" aria-pressed={view === "list"} onClick={() => setView("list")}>List</Button>
      </div>
    </header>
    <label htmlFor="file-search" className="mb-2 block text-sm font-medium">Search by name</label>
    <Input id="file-search" type="search" value={search} onChange={(event) => setSearch(event.target.value)}
      placeholder="Find a file…" aria-describedby="search-scope" />
    <p id="search-scope" className="mb-6 mt-2 text-xs text-muted-foreground">Search filters files loaded so far.</p>
    {query.isPending && <div role="status" aria-label="Loading files" className="grid grid-cols-2 gap-4 sm:grid-cols-3">
      {Array.from({ length: 6 }, (_, i) => <div key={i} aria-hidden="true" className="h-52 animate-pulse rounded-lg bg-secondary motion-reduce:animate-none" />)}
      <span className="sr-only">Loading files…</span>
    </div>}
    {query.error && <div role="alert" className="mb-5 rounded-lg border p-4">
      <p className="mb-3 text-sm text-destructive">Could not load files: {query.error.message}</p>
      <Button variant="outline" disabled={query.isFetching} onClick={() => {
        void (query.isFetchNextPageError ? query.fetchNextPage() : query.refetch());
      }}>Try again</Button>
    </div>}
    {!query.isPending && !query.error && items.length === 0 && <div className="rounded-xl border bg-card p-12 text-center">
      <h2 className="text-xl font-medium">Your space is ready.</h2>
      <p className="mb-6 mt-3 text-sm text-muted-foreground">Upload your first image and find it here.</p>
      <Link to="/" className="inline-flex rounded-full bg-primary px-6 py-3 text-primary-foreground focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ring">Upload a file</Link>
    </div>}
    {visible.length > 0 && <ul aria-label="Your files" className={view === "grid" ? "grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3" : "space-y-3"}>
      {visible.map((file) => <FileEntry key={file.id} file={file} list={view === "list"} />)}
    </ul>}
    {items.length > 0 && visible.length === 0 && <p role="status" className="py-10 text-center text-muted-foreground">No loaded files match this name.</p>}
    <div ref={sentinel} data-testid="files-sentinel" className="mt-6 flex min-h-12 justify-center">
      {hasNextPage && !isFetchNextPageError && <Button variant="outline" disabled={isFetchingNextPage} onClick={() => { void fetchNextPage(); }}>
        {isFetchingNextPage ? "Loading more…" : "Load more files"}
      </Button>}
    </div>
  </section>;
}
