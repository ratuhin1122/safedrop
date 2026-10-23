import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/hooks/use-theme";
import { loadTurnstile } from "@/lib/turnstile";

type Props = { siteKey: string; onToken: (token: string) => void; resetKey: number };

export function Turnstile({ siteKey, onToken, resetKey }: Props) {
  const container = useRef<HTMLDivElement>(null);
  const callback = useRef(onToken);
  callback.current = onToken;
  const { theme } = useTheme();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    if (!siteKey) return;
    let active = true;
    let remove: (() => void) | undefined;
    setError(""); setLoading(true); callback.current("");
    const invalidate = (message: string) => {
      if (active) { callback.current(""); setError(message); setLoading(false); }
    };
    loadTurnstile().then((api) => {
      if (!active || !container.current) return;
      const id = api.render(container.current, {
        sitekey: siteKey, theme, size: "flexible",
        callback: (token) => { if (active) { callback.current(token); setError(""); setLoading(false); } },
        "expired-callback": () => invalidate("Security check expired. Please verify again."),
        "timeout-callback": () => invalidate("Security check timed out. Please try again."),
        "error-callback": () => { invalidate("Security check failed. Please try again."); return true; },
      });
      remove = () => api.remove(id);
      setLoading(false);
    }).catch((failure: unknown) => invalidate(failure instanceof Error ? failure.message : "Security check unavailable."));
    return () => { active = false; remove?.(); };
  }, [siteKey, theme, resetKey, retry]);

  if (!siteKey) return null;
  return <div className="space-y-2" role="group" aria-label="Security verification">
    <div ref={container} className="min-h-16" />
    {loading && <p role="status" className="text-sm text-muted-foreground">Loading security check…</p>}
    {error && <div className="space-y-2">
      <p role="alert" className="text-sm text-destructive">{error}</p>
      <Button type="button" variant="outline" onClick={() => setRetry((value) => value + 1)}>Retry security check</Button>
    </div>}
  </div>;
}
