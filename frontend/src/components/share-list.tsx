import { useEffect, useState } from "react";
import type { ShareItem } from "@/api/schemas";
import { useShareHistory } from "@/hooks/use-share-history";
import { ShareRow } from "@/components/share-row";

export function ShareList({ shares, fileId, names = {} }: {
  shares: ShareItem[]; fileId?: string; names?: Record<string, string>;
}) {
  const { revoked } = useShareHistory();
  const [now, setNow] = useState(Date.now);
  useEffect(() => { const timer = setInterval(() => setNow(Date.now()), 30000); return () => clearInterval(timer); }, []);
  const items = [...new Map([...shares, ...Object.values(revoked).filter((share) => !fileId || share.file_id === fileId)]
    .map((share) => [share.id, share])).values()].sort((a, b) => Date.parse(a.expires_at) - Date.parse(b.expires_at) || a.id.localeCompare(b.id));
  if (!items.length) return <p className="mt-3 text-sm text-muted-foreground">No share links yet.</p>;
  return <ul aria-label="Shared links" className="mt-3 space-y-3">
    {items.map((share) => <ShareRow key={share.id} share={share} now={now} fileName={names[share.file_id]} />)}
  </ul>;
}
