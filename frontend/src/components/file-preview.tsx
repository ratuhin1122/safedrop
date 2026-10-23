import { useEffect, useState } from "react";
import type { FileItem } from "@/api/schemas";
import { variantBlob } from "@/api/variant-media";

export function FilePreview({ file }: { file: FileItem }) {
  const [images, setImages] = useState<{ url: string; width: number }[]>([]);
  const [error, setError] = useState("");
  useEffect(() => {
    const abort = new AbortController();
    const urls: string[] = [];
    const variants = Object.values(file.variants).filter((variant) => variant !== null);
    void Promise.all(variants.map(async (variant) => {
      const blob = await variantBlob(variant.url, abort.signal);
      if (abort.signal.aborted) return null;
      const url = URL.createObjectURL(blob);
      urls.push(url);
      return { url, width: variant.width };
    })).then((values) => { if (!abort.signal.aborted) setImages(values.filter((value) => value !== null)); })
      .catch((error: unknown) => {
        if (!abort.signal.aborted) setError(error instanceof Error ? error.message : "Preview unavailable.");
      });
    return () => { abort.abort(); urls.forEach((url) => URL.revokeObjectURL(url)); };
  }, [file]);
  if (error) return <p role="alert" className="text-sm text-destructive">Could not load preview: {error}</p>;
  if (!images.length) return <p className="py-10 text-center text-sm text-muted-foreground">Preview unavailable.</p>;
  return <img loading="lazy" decoding="async" src={images[0].url} srcSet={images.map((image) => image.url + " " + image.width + "w").join(", ")}
    sizes="(min-width: 640px) 432px, calc(100vw - 48px)" alt={file.original_name}
    className="max-h-80 w-full rounded-lg bg-secondary object-contain" />;
}
