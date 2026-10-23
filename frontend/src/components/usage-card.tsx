import type { Usage } from "@/api/schemas";
import { formatBytes } from "@/lib/upload";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

export const quotaExhausted = (usage?: Usage) => Boolean(usage && usage.bytes_today >= usage.bytes_limit);

type Props = { usage?: Usage; signedIn?: boolean; loading?: boolean; error?: Error | null; onRefresh?: () => void };

export function UsageCard({ usage, signedIn = false, loading = false, error, onRefresh }: Props) {
  const percent = usage ? usage.bytes_limit > 0 ? usage.bytes_today / usage.bytes_limit * 100 : 100 : 0;
  const exhausted = quotaExhausted(usage);
  return <section aria-label="Upload usage" className="mb-6 rounded-lg border bg-background/50 p-4">
    {!signedIn ? <p className="text-sm text-muted-foreground">Sign in to see your usage.</p>
      : !usage && loading ? <p className="text-sm text-muted-foreground">Loading your usage…</p>
      : usage && <div className="flex flex-wrap items-center gap-4">
        <div role="progressbar" aria-label="Daily byte usage" aria-valuemin={0} aria-valuemax={100}
          aria-valuenow={Math.min(100, Math.round(percent))}
          aria-valuetext={`${formatBytes(usage.bytes_today)} of ${formatBytes(usage.bytes_limit)}`}
          className={cn("relative grid size-16 shrink-0 place-items-center text-primary",
            percent > 80 && "text-warning", exhausted && "text-destructive")}>
          <svg viewBox="0 0 40 40" className="absolute inset-0 size-full -rotate-90" aria-hidden="true">
            <circle cx="20" cy="20" r="16" fill="none" stroke="var(--border)" strokeWidth="3" />
            <circle cx="20" cy="20" r="16" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round"
              pathLength="100" strokeDasharray="100" strokeDashoffset={100 - Math.min(100, percent)} />
          </svg>
          <span className="text-xs font-semibold">{Math.min(100, Math.round(percent))}%</span>
        </div>
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-medium">Daily allowance</h2>
          <p className="mt-1 text-sm text-muted-foreground">{formatBytes(usage.bytes_today)} / {formatBytes(usage.bytes_limit)}</p>
          {exhausted ? <p className="mt-1 text-xs text-destructive">Daily allowance used up. Resets at midnight UTC.</p>
            : percent > 80 && <p className="mt-1 text-xs text-warning">Over 80% of your daily allowance used.</p>}
        </div>
        <div className="min-w-0 text-sm">
          <h3 className="text-xs text-muted-foreground">Uploads this hour</h3>
          <p className="mt-1 font-medium">{usage.uploads_this_hour} / {usage.uploads_limit}</p>
          <p className="mt-1 max-w-36 text-xs text-muted-foreground">Limit applies per configured upload window.</p>
        </div>
      </div>}
    {error && signedIn && <div className="mt-2 space-y-2">
      <p role="alert" className="text-sm text-destructive">Could not load usage: {error.message}</p>
      <Button variant="outline" onClick={onRefresh}>Refresh usage</Button>
    </div>}
  </section>;
}
