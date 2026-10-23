import { useMemo } from "react";
import { UploadDropZone } from "@/components/upload-drop-zone";
import { UploadQueue } from "@/components/upload-queue";
import { useUploadQueue } from "@/hooks/use-upload-queue";
import { uploadLimit } from "@/lib/upload";

export function UploadPage() {
  const config = useMemo(() => {
    try { return { maxBytes: uploadLimit(), error: null }; }
    catch (error) { return { maxBytes: 0, error: error instanceof Error ? error.message : "Upload configuration is invalid." }; }
  }, []);
  const { items, addFiles, removeFile } = useUploadQueue(config.maxBytes);
  return <section className="mx-auto max-w-2xl rounded-xl border bg-card p-5 sm:p-10" aria-labelledby="upload-title">
    <h1 id="upload-title" className="text-3xl font-semibold tracking-tight">A little space to share.</h1>
    <p className="mb-8 mt-3 text-sm leading-relaxed text-muted-foreground">Choose the images you want to send.</p>
    {config.error ? <p role="alert" className="text-sm text-destructive">{config.error}</p> : <>
      <UploadDropZone maxBytes={config.maxBytes} onFiles={addFiles} />
      <p id="upload-privacy" className="mt-5 text-center text-xs leading-relaxed text-muted-foreground">
        Files stay in this tab. Nothing is uploaded yet.<br />
        Extensions are a quick hint; the server will verify file contents when uploads are enabled.
      </p>
      <UploadQueue items={items} onRemove={removeFile} />
    </>}
  </section>;
}
