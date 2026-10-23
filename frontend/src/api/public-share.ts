import { z } from "zod";
import { apiRequest } from "@/api/client";
import { shareTokenSchema } from "@/api/schemas";

export const publicShareSchema = z.object({
  name: z.string(), mime: z.string(), size: z.number().int().nonnegative(),
  expires_at: z.iso.datetime({ offset: true }),
});
export type PublicShare = z.infer<typeof publicShareSchema>;
export function getPublicShare(token: string, password?: string, signal?: AbortSignal) {
  const headers = new Headers();
  if (password !== undefined) {
    const bytes = new TextEncoder().encode(":" + password);
    headers.set("Authorization", "Basic " + btoa(Array.from(bytes, (byte) => String.fromCharCode(byte)).join("")));
  }
  return apiRequest("/s/" + shareTokenSchema.parse(token) + "?info=1", publicShareSchema,
    { auth: false, scope: "public", signal, headers, cache: "no-store" });
}
