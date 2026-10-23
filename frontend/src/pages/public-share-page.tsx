import { useState } from "react";
import { useParams } from "react-router";
import { usePublicShare } from "@/hooks/use-public-share";
import { usePublicMeta } from "@/hooks/use-public-meta";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { formatBytes } from "@/lib/upload";
import { expiryWords } from "@/lib/share-form";

function PublicShare({ token }: { token: string }) {
  usePublicMeta();
  const share = usePublicShare(token);
  const [password, setPassword] = useState("");
  return <main className="grid min-h-dvh place-items-center bg-background px-5 py-10">
    <section className="w-full max-w-md rounded-xl border bg-card p-7 text-center sm:p-10" aria-live="polite" aria-busy={share.busy}>
      <p className="mb-8 text-sm font-medium text-primary">SafeDrop</p>
      {share.screen === "loading" && <p role="status">Opening your shared file…</p>}
      {share.screen === "missing" && <><h1 className="text-2xl font-semibold">Link not found</h1>
        <p className="mt-4 text-sm text-muted-foreground">Check that you have the complete link, or ask the sender for a new one.</p></>}
      {share.screen === "gone" && <><h1 className="text-2xl font-semibold">This link has expired or reached its download limit</h1>
        <p className="mt-4 text-sm text-muted-foreground">Ask the sender for a fresh link.</p></>}
      {share.screen === "password" && <><h1 className="text-2xl font-semibold">This file is password protected</h1>
        <p className="mt-3 text-sm text-muted-foreground">Enter the password the sender shared with you.</p>
        <form className="mt-6 space-y-4 text-left" onSubmit={(event) => {
          event.preventDefault(); const secret = password; setPassword(""); void share.load(secret);
        }}>
          <label htmlFor="public-password" className="block text-sm font-medium">Password</label>
          <Input id="public-password" type="password" autoComplete="off" autoFocus required value={password}
            onChange={(event) => setPassword(event.target.value)} aria-describedby="password-error" />
          {share.error && <p id="password-error" role="alert" className="text-sm text-destructive">{share.error}</p>}
          <Button className="w-full" disabled={share.busy || !password} type="submit">Unlock file</Button>
        </form></>}
      {share.screen === "ready" && share.file && <>
        <div aria-hidden="true" className="mx-auto mb-6 grid size-24 place-items-center rounded-lg bg-secondary text-primary">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="size-12"><path d="M6 3h8l4 4v14H6zM14 3v5h4M9 12h6M9 16h6" /></svg>
        </div>
        <h1 className="break-all text-2xl font-semibold">{share.file.name}</h1>
        <p className="mt-3 text-sm text-muted-foreground">{formatBytes(share.file.size)}</p>
        <p className="mb-7 mt-2 text-sm text-muted-foreground">{expiryWords(share.file.expires_at)}</p>
        <Button className="min-h-14 w-full text-base" disabled={share.busy} onClick={() => { void share.download(); }}>
          {share.busy ? "Downloading…" : "Download"}</Button>
      </>}
      {share.screen === "error" && <><h1 className="text-2xl font-semibold">We couldn’t open this file</h1>
        <p role="alert" className="my-5 text-sm text-destructive">{share.error}</p>
        <Button onClick={() => { void share.load(); }}>Try again</Button></>}
    </section>
  </main>;
}
export function PublicSharePage() {
  const { token = "" } = useParams();
  return <PublicShare key={token} token={token} />;
}
