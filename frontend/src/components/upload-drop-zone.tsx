import { useFilePickerShortcut } from "@/hooks/use-file-picker-shortcut";
import { useRef, useState, type DragEvent } from "react";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/icons";
import { acceptedTypesLabel, fileAccept, formatBytes } from "@/lib/upload";
import { cn } from "@/lib/utils";

type Props = { maxBytes: number; onFiles: (files: File[]) => void; disabled?: boolean };

export function UploadDropZone({ maxBytes, onFiles, disabled = false }: Props) {
  const input = useRef<HTMLInputElement>(null);
  useFilePickerShortcut(input, disabled);
  const depth = useRef(0);
  const [dragging, setDragging] = useState(false);

  function enter(event: DragEvent) {
    event.preventDefault();
    if (disabled) return;
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
    if (!disabled) onFiles(Array.from(event.dataTransfer.files));
  }

  return <div role="region" aria-label="File drop zone" aria-disabled={disabled} aria-describedby="upload-limits upload-privacy"
    onDragEnter={enter} onDragLeave={leave} onDrop={drop}
    onDragOver={(event) => { event.preventDefault(); event.dataTransfer.dropEffect = disabled ? "none" : "copy"; }}
    className={cn("flex min-h-80 flex-col items-center justify-center rounded-lg border-2 border-dashed px-5 py-10 text-center transition-colors sm:min-h-96",
      dragging && !disabled ? "border-primary bg-accent" : "border-input bg-background/50", disabled && "border-destructive")}>
    <span className="mb-6 grid size-16 place-items-center rounded-2xl bg-secondary text-secondary-foreground">
      <Icon name="drop" className="size-8" />
    </span>
    <h2 className="text-xl font-medium tracking-tight sm:text-2xl">{disabled ? "Daily limit reached" : dragging ? "Drop your files here" : "Drop something good here"}</h2>
    {disabled && <p role="alert" className="mt-3 text-sm text-destructive">Your daily upload allowance is exhausted. Try again after midnight UTC.</p>}
    <p id="upload-limits" className="mb-6 mt-3 max-w-xs text-sm leading-relaxed text-muted-foreground">
      {acceptedTypesLabel}<br />Max size is {formatBytes(maxBytes)} per file
    </p>
    <input ref={input} type="file" multiple accept={fileAccept} className="sr-only" tabIndex={-1} disabled={disabled}
      aria-label="Choose files" onChange={(event) => {
        if (!disabled) onFiles(Array.from(event.currentTarget.files ?? []));
        event.currentTarget.value = "";
      }} />
    <Button type="button" disabled={disabled} onClick={() => input.current?.click()} aria-describedby="upload-limits picker-shortcut" aria-keyshortcuts="Alt+Shift+U">Browse files</Button>
    <p id="picker-shortcut" className="mt-3 text-xs text-muted-foreground">Shortcut: Alt + Shift + U</p>
    {!disabled && <p className="mt-4 text-xs text-muted-foreground" aria-live="polite">{dragging ? "Release to check these files." : "or drag and drop multiple files"}</p>}
  </div>;
}
