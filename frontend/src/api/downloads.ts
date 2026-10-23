import { apiResponse, ApiError } from "@/api/client";
import { downloadSchema, idSchema, shareTokenSchema, variantWidthSchema, type VariantWidth } from "@/api/schemas";

function basicPassword(password: string) {
  const bytes = new TextEncoder().encode(`:${password}`);
  return `Basic ${btoa(Array.from(bytes, (value) => String.fromCharCode(value)).join(""))}`;
}

async function streamDownload(response: Response) {
  const length = response.headers.get("Content-Length");
  const metadata = downloadSchema.safeParse({
    contentType: response.headers.get("Content-Type") ?? "application/octet-stream",
    contentLength: length === null ? null : Number(length),
    disposition: response.headers.get("Content-Disposition"), url: response.url,
  });
  if (!response.body || !metadata.success) {
    await response.body?.cancel();
    throw new ApiError("invalid_response", "SafeDrop returned an invalid download response.", response.status);
  }
  // The caller streams to a destination or cancels the reader; never buffer files here.
  return { ...metadata.data, stream: response.body };
}

export async function downloadVariant(fileId: string, width: VariantWidth, signal?: AbortSignal) {
  const path = `/files/${idSchema.parse(fileId)}/variants/${variantWidthSchema.parse(width)}`;
  return streamDownload(await apiResponse(path, { signal, headers: { Accept: "*/*" } }));
}

export async function downloadShare(token: string, password?: string, signal?: AbortSignal) {
  const path = `/s/${shareTokenSchema.parse(token)}`;
  const headers: HeadersInit = { Accept: "*/*" };
  if (password !== undefined) headers.Authorization = basicPassword(password);
  return streamDownload(await apiResponse(path, { signal, auth: false, scope: "public", headers }));
}

export type Download = Awaited<ReturnType<typeof downloadVariant>>;
