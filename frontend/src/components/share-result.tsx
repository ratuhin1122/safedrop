import { useEffect, useState } from "react";
import QRCode from "qrcode";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";
import { expiryWords } from "@/lib/share-form";

export function ShareResult({ url, expiresAt }: { url: string; expiresAt: string }) {
  const { toast } = useToast();
  const [qr, setQr] = useState("");
  const [qrError, setQrError] = useState("");
  const [copyError, setCopyError] = useState("");
  const [copying, setCopying] = useState(false);
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    let mounted = true;
    void QRCode.toDataURL(url, { width: 256, margin: 4, errorCorrectionLevel: "M", color: { dark: "#000000", light: "#ffffff" } })
      .then((data) => { if (mounted) setQr(data); })
      .catch(() => { if (mounted) setQrError("Could not generate the QR code. You can still copy the link."); });
    const timer = setInterval(() => setNow(Date.now()), 60000);
    return () => { mounted = false; clearInterval(timer); };
  }, [url]);
  async function copy() {
    setCopyError(""); setCopying(true);
    try { await navigator.clipboard.writeText(url); toast({ title: "Copied", description: "The share link is on your clipboard." }); }
    catch { setCopyError("Could not copy the link. Select the URL and copy it manually."); }
    finally { setCopying(false); }
  }
  return <div className="space-y-5">
    <p role="status" className="text-sm text-muted-foreground">{expiryWords(expiresAt, now)}</p>
    <p className="text-xs text-muted-foreground">Expires {new Date(expiresAt).toLocaleString()}</p>
    <label htmlFor="generated-share-url" className="block text-sm font-medium">Your share link</label>
    <Input id="generated-share-url" readOnly value={url} onFocus={(event) => event.currentTarget.select()} />
    <Button className="w-full" disabled={copying} onClick={() => { void copy(); }}>{copying ? "Copying…" : "Copy link"}</Button>
    {copyError && <p role="alert" className="text-sm text-destructive">{copyError}</p>}
    {qr ? <img src={qr} width={256} height={256} alt="QR code for your share link" className="mx-auto max-w-full rounded-md" />
      : <p className="text-center text-sm text-muted-foreground">{qrError || "Creating QR code…"}</p>}
    <p className="text-center text-xs text-muted-foreground">Anyone with this link can access the file, subject to your limits.</p>
  </div>;
}
