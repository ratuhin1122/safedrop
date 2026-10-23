import { useRef, useState, type DragEvent } from "react";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/icons";
import { acceptedTypesLabel, fileAccept, formatBytes } from "@/lib/upload";
import { cn } from "@/lib/utils";

type Props = { maxBytes: number; onFiles: (files: File[]) => void };

export function UploadDropZone({ maxBytes, onFiles }: Props) {
  const input = useRef<HTMLInputElement>(null);
  const depth = useRef(0);
  const [dragging, setDragging] = useState(false);

  function enter(event: DragEvent) {
    event.preventDefault();
    if (!Array.from(event.dataTransfer.types).includes("Files")) return;
    depth.current += 1;
    setDragging(true);
  }
  function leave(event: DragEvent) {
    event.preventDefault();
    depth.current = Math.max(0, depth.current - 1);
    if (depth.current === 0) setDragging(false);
  }
  function drop(event: DragEvent) {
    event.preventDefault();
    depth.current = 0;
    setDragging(false);
    onFiles(Array.from(event.dataTransfer.files));
  }

  return <div role="region" aria-label="File drop zone" aria-describedby="upload-limits upload-privacy"
    onDragEnter={enter} onDragLeave={leave} onDrop={drop}
    onDragOver={(event) => { event.preventDefault(); event.dataTransfer.dropEffect = "copy"; }}
    className={cn("flex min-h-80 flex-col items-center justify-center rounded-lg border-2 border-dashed px-5 py-10 text-center transition-colors sm:min-h-96",
      dragging ? "border-primary bg-accent" : "border-input bg-background/50")}>
    <span className="mb-6 grid size-16 place-items-center rounded-2xl bg-secondary text-secondary-foreground">
      <Icon name="drop" className="size-8" />
    </span>
    <h2 className="text-xl font-medium tracking-tight sm:text-2xl">{dragging ? "Drop your files here" : "Drop something good here"}</h2>
    <p id="upload-limits" className="mb-6 mt-3 max-w-xs text-sm leading-relaxed text-muted-foreground">
      {acceptedTypesLabel}<br />Max size is {formatBytes(maxBytes)} per file
    </p>
    <input ref={input} type="file" multiple accept={fileAccept} className="sr-only" tabIndex={-1}
      aria-label="Choose files" onChange={(event) => {
        onFiles(Array.from(event.currentTarget.files ?? []));
        event.currentTarget.value = "";
      }} />
    <Button type="button" onClick={() => input.current?.click()} aria-describedby="upload-limits">Browse files</Button>
    <p className="mt-4 text-xs text-muted-foreground" aria-live="polite">{dragging ? "Release to check these files." : "or drag and drop multiple files"}</p>
  </div>;
}
