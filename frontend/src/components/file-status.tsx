import { useFileStatus } from "@/hooks/use-file-status";
import { FileStepper } from "@/components/file-stepper";
import { Button } from "@/components/ui/button";
import { stepLabels } from "@/lib/file-status";

export function FileStatus({ id, name }: { id: string; name: string }) {
  const query = useFileStatus(id);
  const state = query.data?.status ?? "uploaded";
  const summary = state === "uploaded" ? "Uploaded, checking file..."
    : state === "rejected" ? "File rejected" : state === "ready" ? "Ready to share" : `${stepLabels[state]} your file…`;
  return <div aria-label={`Processing status for ${name}`}>
    <p role="status" className="mt-2 text-sm text-primary"><span className="sr-only">{name}: </span>{summary}</p>
    <FileStepper file={query.data ?? { status: "uploaded", reject_reason: null }} />
    {query.paused && !query.complete && <p className="mt-2 text-xs text-muted-foreground">Status updates paused while this tab is hidden.</p>}
    {query.error && !query.complete && <div className="mt-3 space-y-2">
      <p role="alert" className="text-sm text-destructive">Could not check file status: {query.error.message}</p>
      <Button variant="outline" disabled={query.paused || query.isFetching} onClick={() => void query.refetch()}>Check again</Button>
    </div>}
  </div>;
}
