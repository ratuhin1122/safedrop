import { apiRequest, jsonBody } from "@/api/client";
import { createdShareSchema, createShareSchema, emptySchema, idSchema, sharesSchema, type CreateShareInput } from "@/api/schemas";

export function createShare(fileId: string, input: CreateShareInput, signal?: AbortSignal) {
  return apiRequest(`/files/${idSchema.parse(fileId)}/share`, createdShareSchema, {
    method: "POST", signal, ...jsonBody(createShareSchema.parse(input)),
  });
}

export function listShares(fileId: string, signal?: AbortSignal) {
  return apiRequest(`/files/${idSchema.parse(fileId)}/shares`, sharesSchema, { signal });
}

export function deleteShare(id: string, signal?: AbortSignal) {
  return apiRequest(`/shares/${idSchema.parse(id)}`, emptySchema, { method: "DELETE", signal });
}
