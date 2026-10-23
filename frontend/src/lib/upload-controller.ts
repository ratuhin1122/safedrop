import { ApiError } from "@/api/client";
import { uploadWithProgress, type UploadTask } from "@/api/upload-xhr";
import { checkFile } from "@/lib/upload";
import { friendlyUploadError, type QueueItem } from "@/lib/upload-state";

type Snapshot = { items: QueueItem[]; retryAt: number };

export class UploadController {
  private snapshot: Snapshot = { items: [], retryAt: 0 };
  private listeners = new Set<() => void>();
  private active = new Map<number, UploadTask>();
  private nextId = 0;
  private stopped = false;
  private wake?: ReturnType<typeof setTimeout>;
  constructor(private maxBytes: number) {}
  getSnapshot = () => this.snapshot;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  mount = () => { this.stopped = false; };
  private publish(items = this.snapshot.items, retryAt = this.snapshot.retryAt) {
    this.snapshot = { items, retryAt };
    this.listeners.forEach((listener) => listener());
  }
  private patch(id: number, update: Partial<QueueItem>) {
    this.publish(this.snapshot.items.map((item) => item.id === id ? { ...item, ...update } : item));
  }
  addFiles = (files: File[]) => {
    const items = files.map((file): QueueItem => {
      const error = checkFile(file, this.maxBytes);
      return { id: this.nextId++, file, error, state: error ? "invalid" : "selected", progress: 0 };
    });
    this.publish([...this.snapshot.items, ...items]);
  };
  start = () => {
    if (Date.now() < this.snapshot.retryAt) return;
    this.publish(this.snapshot.items.map((item) => item.state === "selected" ? { ...item, state: "queued" } : item));
    this.pump();
  };
  retry = (id: number) => {
    if (Date.now() < this.snapshot.retryAt) return;
    const item = this.snapshot.items.find((item) => item.id === id);
    if (!item || !["error", "cancelled"].includes(item.state)) return;
    this.patch(id, { state: "queued", error: null, errorStatus: undefined, progress: 0 });
    this.pump();
  };
  cancel = (id: number) => {
    this.patch(id, { state: "cancelled", error: null });
    this.active.get(id)?.abort();
  };
  removeFile = (id: number) => {
    this.active.get(id)?.abort();
    this.publish(this.snapshot.items.filter((item) => item.id !== id));
  };
  dispose = () => {
    this.stopped = true;
    clearTimeout(this.wake);
    this.active.forEach((task) => task.abort());
  };
  private pump() {
    if (this.stopped || Date.now() < this.snapshot.retryAt) return;
    while (this.active.size < 3) {
      const item = this.snapshot.items.find((item) => item.state === "queued" && !this.active.has(item.id));
      if (!item) break;
      this.patch(item.id, { state: "uploading" });
      const task = uploadWithProgress(item.file, (progress) => {
        if (!this.stopped && this.snapshot.items.some((row) => row.id === item.id && row.state === "uploading")) this.patch(item.id, { progress });
      });
      this.active.set(item.id, task);
      task.promise.then((response) => {
        if (this.current(item.id)) this.patch(item.id, { state: "uploaded", progress: 100, fileId: response.id });
      }).catch((error: unknown) => {
        if (!this.current(item.id)) return;
        if (error instanceof ApiError && error.status === 429) {
          const retryAt = Math.max(this.snapshot.retryAt, Date.now() + Math.max(0, error.retryAfterSeconds ?? 60) * 1000);
          this.publish(undefined, retryAt);
          clearTimeout(this.wake);
          this.wake = setTimeout(() => { if (!this.stopped) { this.publish(); this.pump(); } }, Math.min(retryAt - Date.now(), 2147483647));
        }
        this.patch(item.id, { state: "error", error: friendlyUploadError(error), errorStatus: error instanceof ApiError ? error.status : undefined });
      }).finally(() => { this.active.delete(item.id); this.pump(); });
    }
  }
  private current(id: number) {
    return !this.stopped && this.snapshot.items.some((item) => item.id === id && item.state === "uploading");
  }
}
