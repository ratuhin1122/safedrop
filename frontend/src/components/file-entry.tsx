import type { FileItem } from "@/api/schemas";
import { FileThumbnail } from "@/components/file-thumbnail";
import { formatBytes } from "@/lib/upload";
import { cn } from "@/lib/utils";

export function FileEntry({ file, list }: { file: FileItem; list: boolean }) {
  return <li className={cn("overflow-hidden rounded-lg border bg-card", list && "flex items-center gap-4 p-3")}>
    <div className={list ? "size-16 shrink-0 overflow-hidden rounded-md" : "aspect-[4/3]"}>
      <FileThumbnail file={file} />
    </div>
    <div className={cn("min-w-0 flex-1 p-4", list && "p-0")}>
      <h2 className="truncate text-sm font-medium" title={file.original_name}>{file.original_name}</h2>
      <p className="mt-1 text-xs text-muted-foreground">{formatBytes(file.size)} · {new Date(file.created_at).toLocaleDateString()}</p>
      <span className={cn("mt-3 inline-flex rounded-full bg-secondary px-3 py-1 text-xs capitalize",
        file.status === "ready" && "bg-accent text-accent-foreground",
        file.status === "rejected" && "bg-destructive/10 text-destructive")}>{file.status}</span>
      {file.status === "rejected" && <p className="mt-2 text-xs text-destructive">{file.reject_reason || "This file could not be accepted."}</p>}
    </div>
  </li>;
}
