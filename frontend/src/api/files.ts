import { apiRequest } from "@/api/client";
import { emptySchema, filesSchema, fileSchema, idSchema, pageInputSchema, uploadSchema, uploadInputSchema, type PageInput } from "@/api/schemas";

export function uploadFile(file: File, signal?: AbortSignal) {
  uploadInputSchema.parse(file);
  const form = new FormData();
  form.append("file", file, file.name);
  return apiRequest("/files", uploadSchema, { method: "POST", body: form, signal });
}

export function listFiles(input: PageInput = {}, signal?: AbortSignal) {
  const parsed = pageInputSchema.parse(input);
  const params = new URLSearchParams();
  if (parsed.cursor) params.set("cursor", parsed.cursor);
  if (parsed.limit !== undefined) params.set("limit", String(parsed.limit));
  const query = params.size ? `?${params}` : "";
  return apiRequest(`/files${query}`, filesSchema, { signal });
}

export function getFile(id: string, signal?: AbortSignal) {
  return apiRequest(`/files/${idSchema.parse(id)}`, fileSchema, { signal });
}

export function deleteFile(id: string, signal?: AbortSignal) {
  return apiRequest(`/files/${idSchema.parse(id)}`, emptySchema, { method: "DELETE", signal });
}
