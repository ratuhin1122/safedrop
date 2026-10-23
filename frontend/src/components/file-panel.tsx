import { ShareDialog } from "@/components/share-dialog";
import { useState } from "react";
import type { FileItem } from "@/api/schemas";
import { variantBlob } from "@/api/variant-media";
import { FilePreview } from "@/components/file-preview";
import { FileShares } from "@/components/file-shares";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { useDeleteFile } from "@/hooks/use-delete-file";
import { formatBytes } from "@/lib/upload";

export function FilePanel({ file, onClose }: { file: FileItem; onClose: () => void }) {
  const [sharing, setSharing] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState("");
  const remove = useDeleteFile();
  const variant = file.variants.w1200 ?? file.variants.w768 ?? file.variants.w375;
  async function download() {
    if (!variant) return;
    setDownloading(true); setError("");
    try {
      const blob = await variantBlob(variant.url);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url; anchor.download = file.original_name.replace(/\.[^.]+$/, "") + (variant.mime === "image/avif" ? ".avif" : ".webp");
      anchor.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) { setError(error instanceof Error ? error.message : "Download failed."); }
    finally { setDownloading(false); }
  }
  return <Modal label="File details" onClose={onClose} sheet>
    <div className="mb-5 flex items-start justify-between gap-3">
      <h2 className="break-all text-xl font-semibold">{file.original_name}</h2>
      <Button variant="ghost" onClick={onClose} aria-label="Close file details">Close</Button>
    </div>
    {file.status === "ready" && <FilePreview file={file} />}
    <p className="my-4 text-sm capitalize">Status: {file.status}</p>
    {file.status === "rejected" ? <p role="alert" className="text-sm text-destructive">{file.reject_reason || "This file was rejected."}</p>
      : <dl className="grid grid-cols-[auto_1fr] gap-x-5 gap-y-2 break-words text-sm">
        <dt>Size</dt><dd>{formatBytes(file.size)}</dd>
        <dt>Dimensions</dt><dd>{file.width && file.height ? file.width + " × " + file.height + " px" : "Not available yet"}</dd>
        <dt>Type</dt><dd>{file.mime}</dd>
        <dt>Created</dt><dd>{new Date(file.created_at).toLocaleString()}</dd>
      </dl>}
    {error && <p role="alert" className="mt-4 text-sm text-destructive">{error}</p>}
    <div className="mt-6 flex flex-wrap gap-3">
      {file.status === "ready" && variant && <Button disabled={downloading} onClick={() => { void download(); }}>
        {downloading ? "Downloading…" : "Download"}</Button>}
      {file.status === "ready" && <Button onClick={() => setSharing(true)}>Share</Button>}
      <Button variant="outline" className="text-destructive" onClick={() => setConfirm(true)}>Delete file</Button>
    </div>
    {file.status !== "rejected" && <FileShares fileId={file.id} />}
    {sharing && <ShareDialog fileId={file.id} onClose={() => setSharing(false)} />}
    {confirm && <Modal label="Confirm deletion" onClose={() => setConfirm(false)}>
      <h2 className="text-xl font-semibold">Delete this file?</h2>
      <p className="my-4 text-sm text-muted-foreground">Its share links will stop working. You can undo while the notification is open.</p>
      <div className="flex flex-wrap gap-3">
        <Button variant="outline" autoFocus onClick={() => setConfirm(false)}>Keep file</Button>
        <Button disabled={deleting} className="bg-destructive text-white" onClick={() => {
          setDeleting(true);
          void remove(file).then(onClose).catch((error: unknown) => { setDeleting(false); setError(error instanceof Error ? error.message : "Could not delete file."); });
        }}>Confirm delete</Button>
      </div>
    </Modal>}
  </Modal>;
}
