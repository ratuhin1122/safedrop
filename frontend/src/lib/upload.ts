import { z } from "zod";

export const acceptedExtensions = ["jpg", "jpeg", "png", "webp", "heic"] as const;
export const fileAccept = acceptedExtensions.map((extension) => `.${extension}`).join(",");
export const acceptedTypesLabel = "JPEG, PNG, WebP and HEIC";
const limitSchema = z.coerce.number().int().positive().max(Number.MAX_SAFE_INTEGER);

export function uploadLimit() {
  const parsed = limitSchema.safeParse(import.meta.env.VITE_MAX_UPLOAD_BYTES ?? 26214400);
  if (!parsed.success) throw new Error("VITE_MAX_UPLOAD_BYTES must be a positive whole number of bytes.");
  return parsed.data;
}

// MB is displayed in 1024-based units to match the API's 25 MiB default.
export function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  const unit = bytes >= 1024 ** 3 ? "GB" : bytes >= 1024 ** 2 ? "MB" : "KB";
  const divisor = unit === "GB" ? 1024 ** 3 : unit === "MB" ? 1024 ** 2 : 1024;
  return `${new Intl.NumberFormat("en", { maximumFractionDigits: 1 }).format(bytes / divisor)} ${unit}`;
}

export function checkFile(file: File, maxBytes: number): string | null {
  if (file.size > maxBytes) return `Max size is ${formatBytes(maxBytes)}`;
  const extension = file.name.split(".").pop()?.toLowerCase();
  if (!file.name.includes(".") || !acceptedExtensions.some((item) => item === extension)) {
    return `Unsupported file extension. Choose ${acceptedTypesLabel}.`;
  }
  if (file.size === 0) return "This file is empty. Choose a file with content.";
  return null;
}
