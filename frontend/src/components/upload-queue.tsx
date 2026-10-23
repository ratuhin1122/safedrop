import { Button } from "@/components/ui/button";
import { Icon } from "@/components/icons";
import { formatBytes } from "@/lib/upload";
import type { QueueItem } from "@/hooks/use-upload-queue";

export function UploadQueue({ items, onRemove }: { items: QueueItem[]; onRemove: (id: number) => void }) {
  if (!items.length) return null;
  const rejected = items.filter((item) => item.error).length;
  return <section aria-labelledby="queue-title" className="mt-8">
    <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h2 id="queue-title" className="font-medium">Your selection</h2>
      <p role="status" className="text-sm text-muted-foreground">{items.length - rejected} selected{rejected > 0 ? ` · ${rejected} not added` : ""}</p>
    </div>
    <ul className="space-y-3" aria-label="File queue">
      {items.map(({ id, file, error }) => <li key={id} className="flex items-center gap-3 rounded-lg border bg-background/50 p-4">
        <div className="min-w-0 flex-1">
          <p className="break-all text-sm font-medium">{file.name}</p>
          <p className="mt-1 text-xs text-muted-foreground">{formatBytes(file.size)} · {error ? "Not added" : "Selected"}</p>
          {error && <p role="alert" className="mt-2 text-sm text-destructive"><span className="sr-only">{file.name}: </span>{error}</p>}
        </div>
        <Button type="button" variant="ghost" size="icon" aria-label={`Remove ${file.name}`} onClick={() => onRemove(id)}>
          <Icon name="close" className="size-4" />
        </Button>
      </li>)}
    </ul>
  </section>;
}
