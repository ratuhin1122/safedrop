import type { Usage } from "@/api/schemas";
import { UsageCard, quotaExhausted } from "@/components/usage-card";
import { useEffect, useMemo, useRef } from "react";
import { UploadDropZone } from "@/components/upload-drop-zone";
import { UploadQueue } from "@/components/upload-queue";
import { useUploadQueue } from "@/hooks/use-upload-queue";
import { uploadLimit } from "@/lib/upload";
import { useCountdown } from "@/hooks/use-countdown";
import { Button } from "@/components/ui/button";

type Props = { usage?: Usage; signedIn?: boolean; loading?: boolean; error?: Error | null; onRefresh?: () => void };
export function UploadWorkspace(props: Props) {
  const exhausted = quotaExhausted(props.usage);
  const config = useMemo(() => {
    try { return { maxBytes: uploadLimit(), error: null }; }
    catch (error) { return { maxBytes: 0, error: error instanceof Error ? error.message : "Upload configuration is invalid." }; }
  }, []);
  const { items, addFiles, removeFile, retryAt, start, retry, cancel } = useUploadQueue(config.maxBytes, exhausted);
  const completed = items.filter((item) => item.state === "uploaded").length;
  const previous = useRef(0);
  useEffect(() => { if (completed > previous.current) props.onRefresh?.(); previous.current = completed; }, [completed, props.onRefresh]);
  const cooldown = useCountdown(retryAt);
  const selected = items.filter((item) => item.state === "selected").length;
  return <section className="mx-auto max-w-2xl rounded-xl border bg-card p-5 sm:p-10" aria-labelledby="upload-title">
    <h1 id="upload-title" className="text-3xl font-semibold tracking-tight">A little space to share.</h1>
    <p className="mb-8 mt-3 text-sm leading-relaxed text-muted-foreground">Choose the images you want to send.</p>
    {config.error ? <p role="alert" className="text-sm text-destructive">{config.error}</p> : <>
      <UsageCard {...props} />
      <UploadDropZone disabled={exhausted} maxBytes={config.maxBytes} onFiles={addFiles} />
      <p id="upload-privacy" className="mt-5 text-center text-xs leading-relaxed text-muted-foreground">
        Nothing is sent until you choose Upload files.<br />
        Extensions are a quick hint; SafeDrop checks file contents after upload.
      </p>
      <UploadQueue blocked={exhausted} items={items} cooldown={cooldown} onRemove={removeFile} onRetry={retry} onCancel={cancel} />
      {items.length > 0 && <div className="mt-6 space-y-3">
        {cooldown > 0 && <p role="status" className="text-sm text-destructive">
          Too many uploads, try again in {cooldown} seconds
        </p>}
        <Button className="w-full" disabled={exhausted || selected === 0 || cooldown > 0} onClick={start}>Upload files</Button>
        <p className="text-center text-xs text-muted-foreground">Up to 3 files upload at once. Cancelling cannot undo a completed server upload.</p>
      </div>}
    </>}
  </section>;
}
