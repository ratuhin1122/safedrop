import { useEffect, useState } from "react";
import type { FileItem } from "@/api/schemas";
import { apiResponse } from "@/api/client";
import { getApiBase } from "@/lib/env";

export function FileThumbnail({ file }: { file: FileItem }) {
  const variant = file.status === "ready" ? file.variants.w375 : null;
  const [source, setSource] = useState<{ url: string; source: string }>();
  useEffect(() => {
    if (!variant) return;
    const abort = new AbortController();
    let objectUrl: string | undefined;
    async function load() {
      // Metadata URLs must resolve inside the configured API before attaching the JWT.
      const base = new URL(getApiBase() + "/", window.location.origin);
      const target = new URL(variant!.url, window.location.origin);
      if (target.origin !== base.origin || !target.pathname.startsWith(base.pathname)) return;
      const path = target.pathname.slice(base.pathname.length - 1) + target.search;
      try {
        const response = await apiResponse(path, { signal: abort.signal, headers: { Accept: "image/webp" } });
        const blob = await response.blob();
        if (abort.signal.aborted) return;
        objectUrl = URL.createObjectURL(blob);
        setSource({ url: objectUrl, source: variant!.url });
      } catch { /* The filename and status remain visible when a preview is unavailable. */ }
    }
    void load();
    return () => { abort.abort(); if (objectUrl) URL.revokeObjectURL(objectUrl); };
  }, [variant]);
  return <div className="grid h-full min-h-16 w-full place-items-center overflow-hidden bg-secondary text-muted-foreground">
    {variant && source?.source === variant.url
      ? <img src={source.url} alt="" loading="lazy" className="h-full w-full object-cover" />
      : <span className="px-2 text-center text-xs">{file.status === "ready" ? "No preview" : "Preview pending"}</span>}
  </div>;
}
