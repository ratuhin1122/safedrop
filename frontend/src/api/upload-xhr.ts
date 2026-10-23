import { ApiError } from "@/api/client";
import { errorSchema, uploadSchema, type UploadResponse } from "@/api/schemas";
import { getAccessToken, handleUnauthorized } from "@/api/session";
import { getApiBase } from "@/lib/env";

export type UploadTask = { promise: Promise<UploadResponse>; abort: () => void };

export function uploadWithProgress(file: File, onProgress: (percent: number) => void): UploadTask {
  const xhr = new XMLHttpRequest();
  const promise = new Promise<UploadResponse>((resolve, reject) => {
    xhr.open("POST", `${getApiBase()}/files`);
    xhr.setRequestHeader("Accept", "application/json");
    const token = getAccessToken();
    if (token) xhr.setRequestHeader("Authorization", `Bearer ${token}`);
    xhr.timeout = 120000;
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable && event.total > 0) onProgress(Math.min(100, Math.round(event.loaded / event.total * 100)));
    };
    xhr.onload = () => {
      let payload: unknown;
      try { payload = JSON.parse(xhr.responseText); } catch { payload = undefined; }
      if (xhr.status === 201) {
        const parsed = uploadSchema.safeParse(payload);
        if (parsed.success) resolve(parsed.data);
        else reject(new ApiError("invalid_response", "SafeDrop returned an invalid upload response.", 201));
        return;
      }
      const parsed = errorSchema.safeParse(payload);
      const error = new ApiError(parsed.success ? parsed.data.code : "upload_failed",
        parsed.success ? parsed.data.message : `Upload failed (HTTP ${xhr.status}).`, xhr.status,
        xhr.status === 429 ? xhr.getResponseHeader("Retry-After") : null);
      if (xhr.status === 401) handleUnauthorized();
      reject(error);
    };
    xhr.onerror = () => reject(new ApiError("network_error", "Connection lost. Check your connection and retry.", 0));
    xhr.ontimeout = () => reject(new ApiError("timeout", "Upload timed out. Please retry.", 408));
    xhr.onabort = () => reject(new DOMException("Upload cancelled.", "AbortError"));
    const body = new FormData();
    body.append("file", file, file.name);
    xhr.send(body);
  });
  return { promise, abort: () => xhr.abort() };
}
