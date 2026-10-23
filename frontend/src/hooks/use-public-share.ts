import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "@/api/client";
import { getPublicShare, type PublicShare } from "@/api/public-share";
import { downloadShare } from "@/api/downloads";
import { shareTokenSchema } from "@/api/schemas";

type State = { screen: "loading" | "ready" | "password" | "gone" | "missing" | "error"; file?: PublicShare; error?: string };
export function usePublicShare(token: string) {
  const [state, setState] = useState<State>({ screen: "loading" });
  const [busy, setBusy] = useState(false);
  const controller = useRef<AbortController | null>(null);
  const password = useRef<string | undefined>(undefined);
  const failure = useCallback((error: unknown) => {
    if (error instanceof ApiError) {
      if (error.status === 401) { setState({ screen: "password", error: password.current ? "That password didn't work. Please try again." : undefined }); return; }
      if (error.status === 410) { setState({ screen: "gone" }); return; }
      if (error.status === 404) { setState({ screen: "missing" }); return; }
    }
    setState({ screen: "error", error: error instanceof Error ? error.message : "Could not open this link. Please try again." });
  }, []);
  const load = useCallback(async (secret?: string) => {
    controller.current?.abort();
    const abort = new AbortController(); controller.current = abort;
    password.current = secret; setBusy(true); setState({ screen: "loading" });
    if (!shareTokenSchema.safeParse(token).success) { setState({ screen: "missing" }); setBusy(false); return; }
    try {
      const file = await getPublicShare(token, secret, abort.signal);
      if (!abort.signal.aborted) setState({ screen: "ready", file });
    } catch (error) { if (!abort.signal.aborted) failure(error); }
    finally { if (!abort.signal.aborted) setBusy(false); }
  }, [token, failure]);
  useEffect(() => { void load(); return () => { controller.current?.abort(); password.current = undefined; }; }, [load]);
  async function download() {
    const file = state.file;
    if (!file || busy) return;
    const abort = new AbortController(); controller.current = abort;
    setBusy(true);
    try {
      const result = await downloadShare(token, password.current, abort.signal);
      const blob = await new Response(result.stream).blob();
      if (abort.signal.aborted) return;
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url; anchor.download = file.name; anchor.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) { if (!abort.signal.aborted) failure(error); }
    finally { if (!abort.signal.aborted) setBusy(false); }
  }
  return { ...state, busy, load, download };
}
