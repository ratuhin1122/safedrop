import { z } from "zod";
import type { CreateShareInput } from "@/api/schemas";

export const shareFormSchema = z.object({
  preset: z.enum(["hour", "day", "week", "custom"]),
  customDate: z.string(),
  password: z.string().refine((value) => new TextEncoder().encode(value).length <= 72, "Password must be at most 72 UTF-8 bytes."),
  maxDownloads: z.string().refine((value) => value === "" || (/^[1-9]\d*$/.test(value) && Number.isSafeInteger(Number(value))),
    "Enter a positive whole number of downloads."),
}).superRefine((value, ctx) => {
  if (value.preset === "custom" && !(Date.parse(value.customDate) > Date.now())) {
    ctx.addIssue({ code: "custom", path: ["customDate"], message: "Choose a date and time in the future." });
  }
});
export type ShareFormValues = z.infer<typeof shareFormSchema>;
export function shareInput(values: ShareFormValues): CreateShareInput {
  const hours = { hour: 1, day: 24, week: 168 };
  return {
    expires_at: new Date(values.preset === "custom" ? Date.parse(values.customDate)
      : Date.now() + hours[values.preset] * 3600000).toISOString(),
    ...(values.password ? { password: values.password } : {}),
    ...(values.maxDownloads ? { max_downloads: Number(values.maxDownloads) } : {}),
  };
}
export function expiryWords(expiresAt: string, now = Date.now()) {
  const seconds = Math.ceil((Date.parse(expiresAt) - now) / 1000);
  if (seconds <= 0) return "Expired";
  const [value, unit] = seconds > 86400 ? [Math.ceil(seconds / 86400), "day"]
    : seconds >= 3600 ? [Math.ceil(seconds / 3600), "hour"] : [Math.max(1, Math.ceil(seconds / 60)), "minute"];
  return "Expires in " + value + " " + unit + (value === 1 ? "" : "s");
}
export function publicShareUrl(value: string) {
  const url = new URL(value, window.location.origin);
  if (!["https:", "http:"].includes(url.protocol) || url.username || url.password) throw new Error("SafeDrop returned an invalid share URL.");
  return url.href;
}
