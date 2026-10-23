import type { ShareItem } from "@/api/schemas";
import { expiryWords } from "@/lib/share-form";

export function shareStatus(share: ShareItem, revoked: boolean, now: number) {
  if (revoked) return "Revoked";
  if (Date.parse(share.expires_at) <= now) return "Expired";
  if (share.max_downloads !== null && share.download_count >= share.max_downloads) return "Limit reached";
  return "Active";
}
export function shareTimeLeft(share: ShareItem, now: number) {
  return expiryWords(share.expires_at, now).replace("Expires in ", "") + (Date.parse(share.expires_at) > now ? " left" : "");
}
