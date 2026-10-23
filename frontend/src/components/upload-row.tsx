import { Button } from "@/components/ui/button";
import { Icon } from "@/components/icons";
import { formatBytes } from "@/lib/upload";
import type { QueueItem } from "@/lib/upload-state";
import { FileStatus } from "@/components/file-status";

type Props = { item: QueueItem; cooldown: number; onRemove: (id: number) => void;
  onCancel: (id: number) => void; onRetry: (id: number) => void };

export function UploadRow({ item, cooldown, onRemove, onCancel, onRetry }: Props) {
  const { id, file, error, state, progress } = item;
  const pending = state === "uploading" || state === "queued";
  const message = item.errorStatus === 429 && cooldown > 0
    ? `Too many uploads, try again in ${cooldown} seconds` : error;
  const label = { selected: "Selected", invalid: "Not added", queued: "Waiting to upload",
    uploading: progress === 100 ? "Finishing upload…" : `Uploading · ${progress}%`,
    uploaded: "Uploaded", error: "Upload failed", cancelled: "Cancelled" }[state];

  return <li className="rounded-lg border bg-background/50 p-4">
    <div className="flex items-center gap-3">
      <div className="min-w-0 flex-1">
        <p className="break-all text-sm font-medium">{file.name}</p>
        <p className="mt-1 text-xs text-muted-foreground">{formatBytes(file.size)} · {label}</p>
      </div>
      {pending ? <Button variant="ghost" onClick={() => onCancel(id)} aria-label={`Cancel ${file.name}`}>Cancel</Button>
        : <>
          {(state === "error" || state === "cancelled") && <Button variant="outline" disabled={cooldown > 0}
            onClick={() => onRetry(id)} aria-label={`Retry ${file.name}`}>Retry</Button>}
          <Button variant="ghost" size="icon" aria-label={`Remove ${file.name}`} onClick={() => onRemove(id)}>
            <Icon name="close" className="size-4" />
          </Button>
        </>}
    </div>
    {state === "uploading" && <div className="mt-3">
      <progress max={100} value={progress} aria-label={`Upload progress for ${file.name}`}
        className="h-2 w-full overflow-hidden rounded-full accent-primary" />
    </div>}
    {message && <p role="alert" className="mt-2 text-sm text-destructive"><span className="sr-only">{file.name}: </span>{message}</p>}
    {state === "uploaded" && item.fileId && <FileStatus id={item.fileId} name={file.name} />}
  </li>;
}
