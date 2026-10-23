import { z } from "zod";
import { registerSchema } from "@/api/schemas";

export const loginFormSchema = z.object({
  email: z.email("Enter a valid email address."),
  password: z.string().min(1, "Enter your password."),
});
export const registerFormSchema = registerSchema.extend({ email: z.email("Enter a valid email address.") });
export type AuthFormValues = z.infer<typeof registerFormSchema>;

export function passwordHint(password: string) {
  const bytes = new TextEncoder().encode(password).length;
  if (bytes > 1024) return "Too long — use at most 1024 UTF-8 bytes.";
  if (!password) return "Use a long, unique passphrase (at least 12 UTF-8 bytes).";
  if (bytes < 12) return "Too short — add more characters (at least 12 UTF-8 bytes).";
  if (password.length < 16) return "Length accepted. A longer, unique passphrase is harder to guess.";
  return "Good length. Keep this passphrase unique to SafeDrop.";
}

export function returnPath(state: unknown) {
  if (!state || typeof state !== "object" || !("from" in state)) return "/files";
  const from = state.from;
  return typeof from === "string" && /^\/(?!\/)/.test(from) && !/[\\\r\n]/.test(from)
    && !/^\/(login|register)(?:[/?#]|$)/.test(from) ? from : "/files";
}
