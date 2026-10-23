import { useEffect, useMemo, useSyncExternalStore } from "react";
import { UploadController } from "@/lib/upload-controller";
export type { QueueItem } from "@/lib/upload-state";

export function useUploadQueue(maxBytes: number, blocked = false) {
  const controller = useMemo(() => new UploadController(maxBytes), [maxBytes]);
  const snapshot = useSyncExternalStore(controller.subscribe, controller.getSnapshot, controller.getSnapshot);
  useEffect(() => { controller.mount(); return controller.dispose; }, [controller]);
  useEffect(() => controller.setBlocked(blocked), [controller, blocked]);
  return { ...snapshot, addFiles: controller.addFiles, removeFile: controller.removeFile,
    start: controller.start, retry: controller.retry, cancel: controller.cancel };
}
