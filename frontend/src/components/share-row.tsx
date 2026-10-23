import { useState } from "react";
import type { ShareItem } from "@/api/schemas";
import { deleteShare } from "@/api/shares";
import { useShareHistory } from "@/hooks/use-share-history";
import { useToast } from "@/hooks/use-toast";
import { shareStatus, shareTimeLeft } from "@/lib/share-status";
import { Button } from "@/components/ui/button";
import { Modal } from "@/components/ui/modal";

export function ShareRow({ share, now, fileName }: { share: ShareItem; now: number; fileName?: string }) {
  const history = useShareHistory();
  const { toast } = useToast();
  const [confirm, setConfirm] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const revoked = Boolean(history.revoked[share.id]);
  const status = shareStatus(share, revoked, now);
  const url = history.urls[share.id];
  async function revoke() {
    setPending(true); setError("");
    try { await deleteShare(share.id); history.markRevoked(share); setConfirm(false); }
    catch (error) { setError(error instanceof Error ? error.message : "Could not revoke this link."); }
    finally { setPending(false); }
  }
  async function copy() {
    if (!url) return;
    setError("");
    try { await navigator.clipboard.writeText(url); toast({ title: "Copied", description: "The share link is on your clipboard." }); }
    catch { setError("Could not copy the link. Check clipboard permissions and try again."); }
  }
  return <li className="rounded-lg border bg-card p-4 text-sm">
    {fileName && <h2 className="mb-2 break-all font-medium">{fileName}</h2>}
    <div className="flex flex-wrap items-center justify-between gap-2">
      <span role="status" className="rounded-full bg-secondary px-3 py-1 text-xs font-medium">{status}</span>
      <span className="text-xs text-muted-foreground">{revoked ? "Access removed" : shareTimeLeft(share, now)}</span>
    </div>
    <p className="mt-3">{share.download_count} / {share.max_downloads ?? "Unlimited"} downloads</p>
    <p className="mt-1 text-xs text-muted-foreground">Expiry: {new Date(share.expires_at).toLocaleString()}
      {share.password_protected && " · Password protected"}</p>
    <div className="mt-3 flex flex-wrap gap-2">
      <Button variant="outline" disabled={!url || revoked} onClick={() => { void copy(); }}>Copy link</Button>
      <Button variant="outline" disabled={revoked || pending} onClick={() => { setError(""); setConfirm(true); }}>Revoke</Button>
    </div>
    {!url && !revoked && <p className="mt-2 text-xs text-muted-foreground">Copy unavailable: the URL is only returned when a link is created.</p>}
    {error && !confirm && <p role="alert" className="mt-2 text-destructive">{error}</p>}
    {confirm && <Modal label="Revoke share link" onClose={() => { if (!pending) setConfirm(false); }}>
      <h2 className="text-xl font-semibold">Revoke this link?</h2>
      <p className="my-4 text-sm text-muted-foreground">Anyone using it will lose access. This cannot be undone.</p>
      {error && <p role="alert" className="mb-4 text-sm text-destructive">{error}</p>}
      <div className="flex flex-wrap gap-3">
        <Button autoFocus variant="outline" disabled={pending} onClick={() => setConfirm(false)}>Keep link</Button>
        <Button disabled={pending} onClick={() => { void revoke(); }}>{pending ? "Revoking…" : "Confirm revoke"}</Button>
      </div>
    </Modal>}
  </li>;
}
