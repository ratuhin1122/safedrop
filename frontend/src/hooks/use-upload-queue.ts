import { useCallback, useRef, useState } from "react";
import { checkFile } from "@/lib/upload";

export type QueueItem = { id: number; file: File; error: string | null };

export function useUploadQueue(maxBytes: number) {
  const [items, setItems] = useState<QueueItem[]>([]);
  const nextId = useRef(0);
  const addFiles = useCallback((files: File[]) => {
    const additions = files.map((file) => ({ id: nextId.current++, file, error: checkFile(file, maxBytes) }));
    setItems((current) => [...current, ...additions]);
  }, [maxBytes]);
  const removeFile = (id: number) => setItems((current) => current.filter((item) => item.id !== id));
  return { items, addFiles, removeFile };
}
