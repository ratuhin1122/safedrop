import { ApiError } from "@/api/client";

export type UploadState = "selected" | "invalid" | "queued" | "uploading" | "uploaded" | "error" | "cancelled";
export type QueueItem = { id: number; file: File; error: string | null; state: UploadState;
  progress: number; errorStatus?: number; fileId?: string };

export function friendlyUploadError(error: unknown) {
  if (error instanceof ApiError) {
    switch (error.status) {
      case 413: return "This file is too large. Choose a smaller file.";
      case 415: return "Unsupported file type. Choose JPEG, PNG, WebP or HEIC.";
      case 422: return "Image dimensions are too large. Choose a smaller image.";
      case 429: return "Too many uploads. Please try again.";
    }
  }
  return error instanceof Error ? error.message : "Upload failed. Please retry.";
}
