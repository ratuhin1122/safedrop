import { expect, it } from "vitest";
import { expiryWords, shareFormSchema, shareInput, publicShareUrl } from "@/lib/share-form";

it.each([["hour", 1], ["day", 24], ["week", 168]] as const)("creates the %s preset", (preset, hours) => {
  const before = Date.now();
  const input = shareInput({ preset, customDate: "", password: "", maxDownloads: "" });
  expect(Date.parse(input.expires_at) - before).toBeCloseTo(hours * 3600000, -2);
});
it("describes expiry plainly", () => {
  const now = Date.now();
  expect(expiryWords(new Date(now + 7 * 86400000).toISOString(), now)).toBe("Expires in 7 days");
  expect(expiryWords(new Date(now + 3600000).toISOString(), now)).toBe("Expires in 1 hour");
  expect(expiryWords(new Date(now).toISOString(), now)).toBe("Expired");
});
it("rejects fractional counts, oversized UTF-8 passwords, and invalid custom dates", () => {
  const base = { preset: "custom", customDate: "invalid", password: "é".repeat(37), maxDownloads: "1.5" };
  expect(shareFormSchema.safeParse(base).success).toBe(false);
});
it("resolves relative public links and rejects unsafe schemes", () => {
  expect(publicShareUrl("/s/test")).toBe(window.location.origin + "/s/test");
  expect(() => publicShareUrl("javascript:alert(1)")).toThrow("invalid share URL");
});
