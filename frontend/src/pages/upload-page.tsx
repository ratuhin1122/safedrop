import { useSyncExternalStore } from "react";
import { getAccessToken, subscribeSession } from "@/api/session";
import { UploadWorkspace } from "@/components/upload-workspace";
import { useUsage } from "@/hooks/use-usage";

function ConnectedUpload() {
  const { data, isPending, error, refresh } = useUsage();
  return <UploadWorkspace signedIn usage={data} loading={isPending} error={error} onRefresh={refresh} />;
}

export function UploadPage() {
  const token = useSyncExternalStore(subscribeSession, getAccessToken, () => null);
  return token ? <ConnectedUpload key={token} /> : <UploadWorkspace />;
}
