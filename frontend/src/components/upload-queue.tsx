import { UploadRow } from "@/components/upload-row";
import type { QueueItem } from "@/lib/upload-state";

type Props = { items: QueueItem[]; cooldown: number; onRemove: (id: number) => void;
  onCancel: (id: number) => void; onRetry: (id: number) => void };

export function UploadQueue({ items, cooldown, onRemove, onCancel, onRetry }: Props) {
  if (!items.length) return null;
  const rejected = items.filter((item) => item.state === "invalid").length;
  return <section aria-labelledby="queue-title" className="mt-8">
    <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h2 id="queue-title" className="font-medium">Your selection</h2>
      <p role="status" className="text-sm text-muted-foreground">{items.length - rejected} selected{rejected > 0 ? ` · ${rejected} not added` : ""}</p>
    </div>
    <ul className="space-y-3" aria-label="File queue">
      {items.map((item) => <UploadRow key={item.id} item={item} cooldown={cooldown}
        onRemove={onRemove} onCancel={onCancel} onRetry={onRetry} />)}
    </ul>
  </section>;
}
