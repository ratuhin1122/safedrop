import { z } from "zod";

export const idSchema = z.uuid();
const count = z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER);
const dimension = z.number().int().positive();
const timestamp = z.iso.datetime({ offset: true });
export const errorSchema = z.object({ code: z.string().min(1), message: z.string().min(1) });
export const credentialsSchema = z.object({ email: z.email(), password: z.string().min(1) });
export const registerSchema = credentialsSchema.extend({
  password: z.string().refine((s) => {
    const size = new TextEncoder().encode(s).length;
    return size >= 12 && size <= 1024;
  }, "Password must contain 12–1024 UTF-8 bytes."),
  turnstile_token: z.string().optional(),
});
export const authSchema = z.object({ token: z.string().min(1) });
export const fileStatusSchema = z.enum(["uploaded", "scanning", "processing", "ready", "rejected"]);
export const uploadSchema = z.object({ id: idSchema, status: z.literal("uploaded") });
export const uploadInputSchema = z.file();
export const variantSchema = z.object({
  width: dimension, height: dimension, size: count, mime: z.string().min(1), url: z.string().min(1),
});
export const fileSchema = z.object({
  id: idSchema, original_name: z.string(), mime: z.string(), size: count,
  width: dimension.nullable(), height: dimension.nullable(), status: fileStatusSchema,
  created_at: timestamp, reject_reason: z.string().nullable(),
  variants: z.object({ w375: variantSchema.nullable(), w768: variantSchema.nullable(), w1200: variantSchema.nullable() }),
});
export const pageInputSchema = z.object({ cursor: z.string().max(256).optional(), limit: z.number().int().min(1).max(100).optional() });
export const filesSchema = z.object({ items: z.array(fileSchema), next_cursor: z.string() });
export const createShareSchema = z.object({
  expires_at: timestamp.refine((value) => Date.parse(value) > Date.now(), "Choose a future expiry time."),
  password: z.string().refine((value) => new TextEncoder().encode(value).length <= 72, "Share password exceeds 72 UTF-8 bytes.").optional(),
  max_downloads: count.refine((value) => value > 0, "Download limit must be positive.").optional(),
});
export const createdShareSchema = z.object({ url: z.string().min(1), expires_at: timestamp });
export const shareSchema = z.object({
  id: idSchema, file_id: idSchema, expires_at: timestamp, max_downloads: count.nullable(),
  download_count: count, password_protected: z.boolean(),
});
export const sharesSchema = z.object({ items: z.array(shareSchema) });
export const usageSchema = z.object({ bytes_today: count, bytes_limit: count, uploads_this_hour: count, uploads_limit: count });
export const emptySchema = z.undefined();
export const healthSchema = z.object({ status: z.literal("ok") });
export const variantWidthSchema = z.union([z.literal(375), z.literal(768), z.literal(1200)]);
export const shareTokenSchema = z.string().regex(/^[A-Za-z0-9_-]{22}$/, "Invalid share token.");
export const downloadSchema = z.object({
  contentType: z.string(), contentLength: count.nullable(), disposition: z.string().nullable(), url: z.string(),
});

export type Credentials = z.infer<typeof credentialsSchema>;
export type RegisterInput = z.infer<typeof registerSchema>;
export type AuthResponse = z.infer<typeof authSchema>;
export type FileItem = z.infer<typeof fileSchema>;
export type FilePage = z.infer<typeof filesSchema>;
export type UploadResponse = z.infer<typeof uploadSchema>;
export type PageInput = z.infer<typeof pageInputSchema>;
export type CreateShareInput = z.infer<typeof createShareSchema>;
export type CreatedShare = z.infer<typeof createdShareSchema>;
export type ShareItem = z.infer<typeof shareSchema>;
export type Usage = z.infer<typeof usageSchema>;
export type VariantWidth = z.infer<typeof variantWidthSchema>;
