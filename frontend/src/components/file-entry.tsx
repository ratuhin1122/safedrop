import type { FileItem } from "@/api/schemas";
import { FileThumbnail } from "@/components/file-thumbnail";
import { formatBytes } from "@/lib/upload";
import { cn } from "@/lib/utils";

export function FileEntry({ file, list, onOpen }: { file: FileItem; list: boolean; onOpen?: () => void }) {
  return <li className={cn("overflow-hidden rounded-lg border bg-card", list && "flex items-center gap-4 p-3")}>
    <button onClick={onOpen} aria-label={"Preview " + file.original_name} aria-haspopup="dialog" className={cn("block focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring", list ? "size-16 shrink-0 overflow-hidden rounded-md" : "aspect-[4/3] w-full")}>
      <FileThumbnail file={file} />
    </button>
    <div className={cn("min-w-0 flex-1 p-4", list && "p-0")}>
      <h2 className="truncate text-sm font-medium" title={file.original_name}><button onClick={onOpen} className="max-w-full truncate rounded-sm text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring" aria-haspopup="dialog" aria-label={"Open " + file.original_name}>{file.original_name}</button></h2>
      <p className="mt-1 text-xs text-muted-foreground">{formatBytes(file.size)} · {new Date(file.created_at).toLocaleDateString()}</p>
      <span className={cn("mt-3 inline-flex rounded-full bg-secondary px-3 py-1 text-xs capitalize",
        file.status === "ready" && "bg-accent text-accent-foreground",
        file.status === "rejected" && "bg-destructive/10 text-destructive")}>{file.status}</span>
      {file.status === "rejected" && <p className="mt-2 text-xs text-destructive">{file.reject_reason || "This file could not be accepted."}</p>}
    </div>
  </li>;
}
