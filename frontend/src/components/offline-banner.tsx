import { useSyncExternalStore } from "react";

const subscribe = (notify: () => void) => {
  window.addEventListener("online", notify);
  window.addEventListener("offline", notify);
  return () => { window.removeEventListener("online", notify); window.removeEventListener("offline", notify); };
};
export function OfflineBanner() {
  const online = useSyncExternalStore(subscribe, () => navigator.onLine, () => true);
  return <div aria-live="polite" aria-atomic="true">
    {!online && <p className="border-b bg-secondary px-5 py-3 text-center text-sm text-secondary-foreground">
      You’re offline. Check your connection before uploading or downloading.
    </p>}
  </div>;
}
